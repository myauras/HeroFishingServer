package main

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"

	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"matchgame/game"
	"matchgame/packet"
	"net"
	"os"
	"strings"
	"time"

	serverSDK "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// 環境版本
const (
	ENV_DEV     = "Dev"
	ENV_RELEASE = "Release"
)

const ALLOW_PLAYER_NUMBER int = 4
const TIME_UPDATE_INTERVAL_MS = 200 //每X毫秒更新Server時間

var RoomName string
var connectionTokens []string
var EnvVersion string

func main() {
	go signalListen()

	port := flag.String("port", "7654", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	EnvVersion = *flag.String("Version", "Dev", "version setting")
	if ep := os.Getenv("Version"); ep != "" {
		EnvVersion = ep
	}

	s, err := sdk.NewSDK()
	if err != nil {
		log.Errorf("%s Could not connect to sdk: %v.\n", logger.LOG_Main, err)
	}

	roomDataChan := make(chan *game.Room)

	roomInit := false
	var matchmakerPodName string
	var dbMapID string
	// var gsLoadDone *serverSDK.GameServer
	var playerIDs [ALLOW_PLAYER_NUMBER]string
	s.WatchGameServer(func(gs *serverSDK.GameServer) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%s Could not connect to sdk: %v.\n", logger.LOG_Main, err)
				shutdownServer(s)
			}
		}()

		if !roomInit && gs.ObjectMeta.Labels["RoomName"] != "" {
			log.Infof("%s Start room init!", logger.LOG_Main)
			matchmakerPodName = gs.ObjectMeta.Labels["MatchmakerPodName"]
			var pIDs [ALLOW_PLAYER_NUMBER]string
			for i := 0; i < ALLOW_PLAYER_NUMBER; i++ {
				pIDs[i] = gs.ObjectMeta.Labels[fmt.Sprintf("player%d", i)]
				playerIDs[i] = pIDs[i]
			}

			dbMapID = gs.ObjectMeta.Labels["MapID"]
			// roomGameDataSnap, ok := FirebaseFunction.GetRoomGameData(dbMapID)
			// if !ok {
			// 	return
			// }
			// var gameSetting mainGame.GameSetting
			// err = roomGameDataSnap.DataTo(&gameSetting)
			// if err != nil {
			// 	return
			// }
			roomInit = true
			// gsLoadDone = gs
			RoomName = gs.ObjectMeta.Labels["RoomName"]
			log.Infof("%s ==============InitGameRoom==============", logger.LOG_Main)
			log.Infof("%s MatchmakerPodName: %s", logger.LOG_Main, matchmakerPodName)
			log.Infof("%s RoomName: %s", logger.LOG_Main, RoomName)
			log.Infof("%s PlayerIDs: %s", logger.LOG_Main, pIDs)
			// mainGame.CheckOutCheatData(&gameSetting)
			mainGame.InitGameRoom(dbMapID, RoomName, pIDs, playerIDs, nil, roomDataChan, gs.ObjectMeta.Name)
			log.Infof("%s Init Game Room Success", logger.LOG_Main)
		} else {
			if matchmakerPodName != "" && gs.ObjectMeta.Labels["MatchmakerPodName"] != "" && matchmakerPodName != gs.ObjectMeta.Labels["MatchmakerPodName"] {
				// 要改成atlas function版本
				// FirebaseFunction.WriteErrorLog(map[string]interface{}{
				// 	"ErrorID":    "ALLOCATE ERROR",
				// 	"Message":    "Agones has allocate error in parelle.",
				// 	"CreateTime": time.Now(),
				// })
			}
		}
	})

	// Health Ping
	stop := make(chan struct{})
	go doHealth(s, stop)

	// 將此遊戲房伺服器狀態標示為Ready
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	// 等拿到房間資訊後才開啟socket連線
	gameRoom := <-roomDataChan
	log.Infof("%s GameRoom got metaData", logger.LOG_Main)
	close(roomDataChan)

	// 開啟連線
	log.Infof("%s Open TCP&UDP Connection", logger.LOG_Main)
	go openConnectTCP(s, stop, ":"+*port, gameRoom)
	// FirebaseFunction.CreateGameRoomByRoomName(gsLoadDone.Status.Address, gsLoadDone.Status.Ports[0].Port, gsLoadDone.ObjectMeta.Labels["roomName"], playerIDs, dbMapID, gsLoadDone.ObjectMeta.Name)
	go OpenConnectUDP(s, stop, ":"+*port, gameRoom)
	endGameChan := make(chan struct{})
	gameRoom.StartRun(stop, endGameChan)
	select {
	case <-stop:
		// 錯誤發生寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s game stop chan", logger.LOG_Main)
		shutdownServer(s)
		return
	case <-endGameChan:
		// 遊戲房關閉寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s End game chan", logger.LOG_Main)
		delayShutdownServer(60*time.Second, s, stop)
	}
	<-stop
	shutdownServer(s)
}

// 偵測SIGTERM/SIGKILL的終止訊號，偵測到就刪除遊戲房資料並寫log
func signalListen() {
	ctx, _ := signals.NewSigKillContext()
	<-ctx.Done()
	// FirebaseFunction.DeleteGameRoom(documentID)
	log.Infof("%s Exit signal received. Shutting down.", logger.LOG_Main)
	os.Exit(0)
}

// 開啟TCP連線
func openConnectTCP(s *sdk.SDK, stop chan struct{}, address string, gameRoom *mainGame.GameRoom) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectTCP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("%s Could not start tcp server: %v.\n", logger.LOG_Main, err)
	}
	defer ln.Close() // nolint: errcheck

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("%s Unable to accept incoming tcp connection: %v.\n", logger.LOG_Main, err)
			continue
		}
		go handleConnectionTCP(conn, stop, gameRoom)
	}
}

// 開啟UDP連線
func OpenConnectUDP(s *sdk.SDK, stop chan struct{}, address string, gameRoom *mainGame.GameRoom) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectUDP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		log.Errorf("%s Could not start udp server: %v.\n", logger.LOG_Main, err)
	}
	defer conn.Close() // nolint: errcheck
	for {
		b := make([]byte, 1024)
		n, sender, err := conn.ReadFrom(b)
		if err != nil || n <= 0 {
			log.Errorf("%s Could not read from udp stream: %v.\n", logger.LOG_Main, err)
			continue
		}
		txt := strings.TrimSpace(string(b[:n]))
		// log.Infof("%s Received packet from %v: %v", logger.LOG_Main, sender.String(), txt)
		hasToken := false
		for _, t := range connectionTokens {
			log.Infof("%s Connection Tokens : %s", logger.LOG_Main, t)
			if t == txt {
				hasToken = true
			}
		}
		if hasToken {
			// log.Infof("%s Start Update UDP Message", logger.LOG_Main)
			go handleConnectionUDP(conn, stop, sender, gameRoom)
		}
	}
}

// 處理TCP連線封包
func handleConnectionTCP(conn net.Conn, stop chan struct{}, gameRoom *mainGame.GameRoom) {
	remoteAddr := conn.RemoteAddr().String()
	// log.Infof("%s Client %s connected", logger.LOG_Main, conn.RemoteAddr().String())
	defer conn.Close()
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s HandleConnectionTCP error: %v.", logger.LOG_Main, err)
		}
	}()
	isAuth := false
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(1 * time.Minute))
	for {
		select {
		case <-stop:
			// 被強制終止
			return
		default:
		}
		pack, err := packet.ReadPack(decoder)
		if err != nil {
			gameRoom.PlayerLeave(conn)
			return
		}
		log.Infof("%s Receive: %s from %s \n", logger.LOG_Main, pack.CMD, remoteAddr)

		//除了Auth指令進來未驗證都擋掉
		if !isAuth && pack.CMD != packet.AUTH {
			log.Infof("%s UnAuth command", logger.LOG_Main)
			return
		}

		if pack.CMD == packet.AUTH {

			authContent := packet.AuthCMD{}
			if ok := authContent.Parse(pack.Content); !ok {
				log.Errorf("%s Parse AuthCMD Failed", logger.LOG_Main)
				return
			}
			// token, err := FirebaseFunction.VerifyIDToken(authContent.Token)
			if err != nil {
				log.Errorf("%s Verify ID token failed: %v\n", logger.LOG_Main, err)
			} else {
				isAuth = true
				secretKey := generateSecureToken(32)
				err = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_REPLY,
					PackID: pack.PackID,
					Content: &packet.AuthCMD_Reply{
						IsAuth:   true,
						TokenKey: secretKey,
					},
				})
				if err != nil {
					return
				}
				defer removeConnectionToken(secretKey)
				connectionTokens = append(connectionTokens, secretKey)
				gameRoom.PlayerJoin(conn, encoder, decoder, token.UID)
				continue
			}
			err = packet.SendPack(encoder, &packet.Pack{
				CMD:    packet.AUTH_REPLY,
				PackID: pack.PackID,
				ErrMsg: err.Error(),
				Content: &packet.AuthCMD_Reply{
					IsAuth: false,
				},
			})
			if err != nil {
				return
			}
		} else {
			err = gameRoom.HandleMessage(conn, pack, stop)
			if err != nil {
				log.Errorf("%s GameRoom Handle Message Error: %v\n", logger.LOG_Main, err.Error())
				gameRoom.PlayerLeave(conn)
				return
			}
		}
	}
}

// 處理UDP連線封包
func handleConnectionUDP(conn net.PacketConn, stop chan struct{}, addr net.Addr, gameRoom *mainGame.GameRoom) {
	timer := time.NewTicker(TIME_UPDATE_INTERVAL_MS * time.Millisecond)
	for {
		select {
		case <-stop:
			//被強制終止
			return
		case <-timer.C:
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATE_UDP,
				PackID: -1,
				Content: mainGame.UdpUpdatePacket{
					ServerTime: gameRoom.ServerTime,
				},
			})
			if err != nil {
				log.Errorf("%s Error Parse send UDP message. %s", logger.LOG_Main, err.Error())
				continue
			}
			sendData = append(sendData, '\n')
			_, sendErr := conn.WriteTo(sendData, addr)
			if sendErr != nil {
				log.Errorf("%s Error send UDP message. %s", logger.LOG_Main, sendErr.Error())
				continue
			}
		}
	}
}

// 通知Agones關閉server並結束應用程式
func shutdownServer(s *sdk.SDK) {
	log.Infof("%s Shutdown agones server and exit app.", logger.LOG_Main)
	// 通知Agones關閉server
	if err := s.Shutdown(); err != nil {
		log.Errorf("%s Could not call shutdown: %v", logger.LOG_Main, err)
	}
	// 結束應用
	os.Exit(0)
}

// 延遲關閉Agones server
func delayShutdownServer(delay time.Duration, sdk *sdk.SDK, stop chan struct{}) {
	timer1 := time.NewTimer(delay)
	<-timer1.C
	// 通知Agones關閉server
	if err := sdk.Shutdown(); err != nil {
		log.Errorf("%s Could not call shutdown: %v", logger.LOG_Main, err)
	}
	stop <- struct{}{}
}

// 伺服器健康狀態檢測
func doHealth(sdk *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Errorf("%s Could not send health ping: %v", logger.LOG_Main, err)
		}
		select {
		case <-stop:
			log.Infof("%s Health pings stopped", logger.LOG_Main)
			return
		case <-tick:
		}
	}
}

// 產生連線驗證Token
func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// 移除連線驗證Token
func removeConnectionToken(token string) {
	index := -1
	for i, v := range connectionTokens {
		if v == token {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	after := append(connectionTokens[:index], connectionTokens[index+1:]...)
	connectionTokens = after
}
