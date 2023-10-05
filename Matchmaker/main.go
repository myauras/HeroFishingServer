package main

import (
	"encoding/json"
	"flag"
	logger "matchmaker/logger"
	"matchmaker/packet"
	"matchmaker/setting"
	"net"
	"os"
	"time"

	myModule "github.com/AuroScoz/HeroFishingServer/herofishingGoModule"
	log "github.com/sirupsen/logrus"
)

var Env string                    // 環境版本
var SelfPodName string            // K8s上所屬的Pod名稱
var Receptionist RoomReceptionist // 房間接待員

func main() {
	log.Infof("%s ==============MATCHMAKER START==============", logger.LOG_Main)

	// 設定Port
	port := flag.String("port", "32680", "The port to listen to tcp traffic on")
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = &envPort
	}
	log.Infof("%s Port: %s", logger.LOG_Main, *port)

	// 設定環境版本
	Env = *flag.String("Env", "Dev", "Env setting")
	if envEnv := os.Getenv("Env"); envEnv != "" {
		Env = envEnv
	}
	log.Infof("%s Env: %s", logger.LOG_Main, Env)

	// 設定K8s上所屬的Pod名稱
	SelfPodName = *flag.String("MY_POD_NAME", "myPodName", "Pod Name")
	if envSelfPodName := os.Getenv("MY_POD_NAME"); envSelfPodName != "" {
		SelfPodName = envSelfPodName
	}
	// 設定API public Key
	mongoAPIPublicKey := *flag.String("MongoAPIPublicKey", "", "Mongo API public key setting")
	if envMongoAPIPublicKey := os.Getenv("MongoAPIPublicKey"); envMongoAPIPublicKey != "" {
		mongoAPIPublicKey = envMongoAPIPublicKey
	}
	log.Infof("%s MongoAPIPublicKey: %s", logger.LOG_Main, mongoAPIPublicKey)

	// 設定API private Key
	mongoAPIPrivateKey := *flag.String("MongoAPIPrivateKey", "", "Mongo API private key setting")
	if envMongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey"); envMongoAPIPrivateKey != "" {
		mongoAPIPrivateKey = envMongoAPIPrivateKey
	}
	log.Infof("%s MongoAPIPrivateKey: %s", logger.LOG_Main, mongoAPIPrivateKey)

	// 初始化MongoDB設定
	initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey)

	// 偵聽TCP封包
	src := ":" + *port
	tcpListener, err := net.Listen("tcp", src)
	if err != nil {
		log.Errorf("%s Listen error %s.\n", logger.LOG_Main, err.Error())
	}
	defer tcpListener.Close()
	log.Infof("%s TCP server start and listening on %s.\n", logger.LOG_Main, src)

	// 取Loadbalancer分配給此pod的對外IP並寫入資料庫
	setExternalIP()

	// 初始化配房者
	Receptionist.Init()

	// tcp連線
	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Errorf("%s Connection error %s.\n", logger.LOG_Main, err)
		}
		go handleConnectionTCP(conn)
	}
}

// 初始化MongoDB設定
func initMonogo(mongoAPIPublicKey string, mongoAPIPrivateKey string) {
	log.Infof("%s 初始化mongo開始", logger.LOG_Main)
	myModule.mongo.Init(myModule.mongo.InitData{
		Env:           Env,
		APIPublicKey:  mongoAPIPublicKey,
		APIPrivateKey: mongoAPIPrivateKey,
	})
	log.Infof("%s 初始化mongo完成", logger.LOG_Main)
}

// 取Loadbalancer分配給此pod的對外IP並寫入資料庫
func setExternalIP() {
	ip, err := myModule.k8s.GetLoadBalancerExternalIP(myModule.setting.NAMESPACE_MATCHERSERVER, myModule.setting.MATCHMAKER)
	if err != nil {
		log.Errorf("%s GetLoadBalancerExternalIP error: %v.\n", logger.LOG_Main, err)
	}
	log.Infof("%s External IP: %s.\n", logger.LOG_Main, ip)
}

// 處理TCP封包
func handleConnectionTCP(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Infof("%s Client connected from: %s", logger.LOG_Main, remoteAddr)
	defer conn.Close()

	player := roomPlayer{
		id:     "",
		isAuth: false,
		connTCP: ConnectionTCP{
			Conn:    conn,
			Encoder: json.NewEncoder(conn),
			Decoder: json.NewDecoder(conn),
		},
		mapID: "",
		room:  nil,
	}

	go disconnectCheck(&player)

	for {
		pack, err := packet.ReadPack(player.connTCP.Decoder)
		if err != nil {
			return
		}

		log.Infof("%s Receive %s from %s", logger.LOG_Main, pack.CMD, remoteAddr)

		//收到Auth以外的命令如果未驗證就都擋掉
		if !player.isAuth && pack.CMD != packet.AUTH {

			log.WithFields(log.Fields{
				"cmd":     pack.CMD,
				"address": remoteAddr,
			}).Infof("%s UnAuthed CMD", logger.LOG_Main)
			return
		}

		// 封包處理
		switch pack.CMD {
		case packet.AUTH:
			packHandle_Auth(pack, &player)
		case packet.CREATEROOM:
			log.Infof("%s =========CREATEROOM=========", logger.LOG_Main)
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
		log.Errorf("%s Parse AuthCMD failed", logger.LOG_Main)
		return
	}

	// 還沒實作Auth驗證 先直接設定為true
	playerID, authErr := myModule.mongo.PlayerVerify(authContent.Token)
	// 驗證失敗
	if authErr != nil || playerID == "" {
		log.Errorf("%s Player verify failed: %v", logger.LOG_Main, authErr)
		_ = packet.SendPack(player.connTCP.Encoder, &packet.Pack{
			CMD:    packet.AUTH_REPLY,
			PackID: pack.PackID,
			ErrMsg: "Auth toekn驗證失敗",
			Content: &packet.AuthCMD_Reply{
				IsAuth: false,
			},
		})
	}

	// 驗證通過
	log.Infof("%s Player verify success, playerID: %s", logger.LOG_Main, playerID)
	player.isAuth = true
	err := packet.SendPack(player.connTCP.Encoder, &packet.Pack{
		CMD:    packet.AUTH_REPLY,
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

		log.Error("Parse CreateRoomCMD failed")
		return
	}
	//還沒實作DB資料
	player.id = createRoomCMD.CreaterID

	canCreate := true
	if !canCreate {
		packet.SendPack(player.connTCP.Encoder, &packet.Pack{
			CMD:    packet.CREATEROOM_REPLY,
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
	case setting.MATCH_QUICK: // 快速配對
		player.room = Receptionist.JoinRoom(dbMap, player)
		if player.room == nil {

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

// 斷線玩家偵測
func disconnectCheck(p *roomPlayer) {
	timer := time.NewTicker(setting.DISCONNECT_CHECK_INTERVAL_SECS * time.Second)
	for {
		<-timer.C
		if p.room == nil || p.id == "" {
			log.Infof("%s Disconnect IP: %s , because it's life is over", logger.LOG_Main, p.connTCP.Conn.RemoteAddr().String())
			p.connTCP.Conn.Close()
			return
		}
	}
}

// 送創建房間結果封包
func sendCreateRoomCMD_Reply(player roomPlayer, p packet.Pack, log string) error {
	err := packet.SendPack(player.connTCP.Encoder, &packet.Pack{
		CMD:     packet.CREATEROOM_REPLY,
		PackID:  p.PackID,
		Content: &packet.CreateRoomCMD_Reply{},
		ErrMsg:  log,
	})
	return err
}
