package main

import (
	"encoding/json"
	"flag"
	logger "matchmaker/logger"
	"matchmaker/packet"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	CONNECTION_CHECK_CYCLE = 3
)

// 環境版本
const (
	ENV_DEV     = "Dev"
	ENV_RELEASE = "Release"
)

// 配對類型
const (
	MATCH_QUICK = "Quick"
)

// 命令列表
const (
	AUTH             = "Auth"             // 身分驗證
	AUTH_REPLY       = "Auth_Reply"       // 身分驗證回傳
	CREATEROOM       = "CreateRoom"       // 建立房間
	CREATEROOM_REPLY = "CreateRoom_Reply" // 建立房間回傳
)

var EvnVersion string             // 環境版本
var SelfPodName string            // K8s上所屬的Pod名稱
var Receptionist RoomReceptionist // 房間接待員

func main() {
	log.Infof("%s ==============MATCHMAKER START==============", logger.LOG_Main)

	// 設定Port
	port := flag.String("port", "32680", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	log.Infof("%s Port: %s", logger.LOG_Main, *port)

	// 設定環境版本
	EvnVersion = *flag.String("Version", "Dev", "EnvVersion setting")
	if ep := os.Getenv("Version"); ep != "" {
		EvnVersion = ep
	}
	log.Infof("%s EvnVersion: %s", logger.LOG_Main, EvnVersion)

	// 設定K8s上所屬的Pod名稱
	SelfPodName = *flag.String("MY_POD_NAME", "myPodName", "Pod Name")
	if ep := os.Getenv("MY_POD_NAME"); ep != "" {
		SelfPodName = ep
	}

	// 偵聽TCP封包
	src := ":" + *port
	listener, err := net.Listen("tcp", src)
	if err != nil {
		log.Errorf("%s Listen error %s.\n", logger.LOG_Main, err.Error())
	}
	defer listener.Close()
	log.Infof("%s TCP server start and listening on %s.\n", logger.LOG_Main, src)

	Receptionist.Init()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("%s Connection error %s.\n", logger.LOG_Main, err)
		}
		go handleConnectionTCP(conn)
	}
}

// 處理TCP封包
func handleConnectionTCP(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Infof("%s Client connected from: %s", logger.LOG_Main, remoteAddr)
	defer conn.Close()

	player := roomPlayer{
		id:      "",
		isAuth:  false,
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
		mapID:   "",
		room:    nil,
	}

	go checkForceDisconnect(&player)

	for {
		pack, err := packet.ReadPack(player.decoder)
		if err != nil {
			return
		}
		// 寫LOG
		log.Infof("%s Receive %s from %s", logger.LOG_Main, pack.CMD, remoteAddr)

		//收到Auth以外的命令如果未驗證就都擋掉
		if !player.isAuth && pack.CMD != AUTH {
			// 寫LOG
			log.WithFields(log.Fields{
				"cmd":     pack.CMD,
				"address": remoteAddr,
			}).Infof("%s UnAuthed CMD", logger.LOG_Main)
			return
		}

		// 封包處理
		switch pack.CMD {
		case AUTH:
			packHandle_Auth(pack, &player)
		case CREATEROOM:
			packHandle_CreateRoom(pack, &player, remoteAddr)
		default:
			log.Errorf("%s got unknow Pack CMD: %s", logger.LOG_Main, pack.CMD)
			return
		}

	}
}

// 處理封包-帳戶驗證
func packHandle_Auth(pack packet.Pack, player *roomPlayer) {
	authContent := packet.AuthCMD{}
	if ok := authContent.Parse(pack.Content); !ok {
		// 寫LOG
		log.Error("Parse AuthCMD failed")
		return
	}

	// 還沒實作Auth驗證 先直接設定為true
	auth := true
	// 驗證失敗
	if !auth {
		_ = packet.SendPack(player.encoder, &packet.Pack{
			CMD:    AUTH_REPLY,
			PackID: pack.PackID,
			ErrMsg: "Auth toekn驗證失敗",
			Content: &packet.AuthCMD_Reply{
				IsAuth: false,
			},
		})
	}
	// 驗證通過
	player.isAuth = true
	err := packet.SendPack(player.encoder, &packet.Pack{
		CMD:    AUTH_REPLY,
		PackID: pack.PackID,
		Content: &packet.AuthCMD_Reply{
			IsAuth: true,
		},
	})
	if err != nil {
		return
	}
}

// 處理封包-開遊戲房
func packHandle_CreateRoom(pack packet.Pack, player *roomPlayer, remoteAddr string) {
	createRoomCMD := packet.CreateRoomCMD{}
	if ok := createRoomCMD.Parse(pack.Content); !ok {
		// 寫LOG
		log.Error("Parse CreateRoomCMD failed")
		return
	}
	//還沒實作DB資料
	player.id = createRoomCMD.CreaterID

	canCreate := true
	if !canCreate {
		packet.SendPack(player.encoder, &packet.Pack{
			CMD:    CREATEROOM_REPLY,
			PackID: pack.PackID,
			Content: &packet.CreateRoomCMD_Reply{
				GameServerIP:   "",
				GameServerPort: -1,
			},
			ErrMsg: "創建房間失敗原因",
		})
	}

	// 根據DB地圖設定來開遊戲房
	var dbMap dbMapData

	switch dbMap.matchType {
	case MATCH_QUICK: // 快速配對
		player.room = Receptionist.JoinRoom(dbMap, player)
		if player.room == nil {
			// 寫LOG
			log.WithFields(log.Fields{
				"dbMap":  dbMap,
				"player": player,
			}).Errorf("%s Join quick match room failed", logger.LOG_Main)
			// 回送房間建立失敗封包
			sendCreateRoomCMD_Reply(*player, pack, "Join quick match room failed")
			return
		}
		// 建立遊戲房
		player.room.CreateGame()
	default:
		// 寫LOG
		log.WithFields(log.Fields{
			"dbMap.matchType": dbMap.matchType,
			"remoteAddr":      remoteAddr,
		}).Errorf("%s Undefined match type", logger.LOG_Main)

		// 回送房間建立失敗封包
		if err := sendCreateRoomCMD_Reply(*player, pack, "Undefined match type"); err != nil {
			return
		}
	}
}

func checkForceDisconnect(p *roomPlayer) {
	timer := time.NewTicker(CONNECTION_CHECK_CYCLE * time.Minute)
	for {
		<-timer.C
		if p.room == nil || p.id == "" {
			log.Infof("%s Disconnect because it's life is over: %s", logger.LOG_Main, p.conn.RemoteAddr().String())
			p.conn.Close()
			return
		}
	}
}

func sendCreateRoomCMD_Reply(player roomPlayer, p packet.Pack, log string) error {
	err := packet.SendPack(player.encoder, &packet.Pack{
		CMD:     CREATEROOM_REPLY,
		PackID:  p.PackID,
		Content: &packet.CreateRoomCMD_Reply{},
		ErrMsg:  log,
	})
	return err
}
