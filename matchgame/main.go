package main

import (
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	matchgameSetting "matchgame/setting"

	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	mongo "herofishingGoModule/mongo"

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

var connnTokens []string // 連線驗證Token
var Env string           // 環境版本

func main() {
	log.SetOutput(os.Stdout) //設定log輸出方式
	log.Infof("%s ==============MATCHGAME 啟動==============", logger.LOG_Main)
	go signalListen()
	port := flag.String("port", "7654", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	Env = *flag.String("Version", "Dev", "version setting")
	if ep := os.Getenv("Version"); ep != "" {
		Env = ep
	}
	agonesSDK, err := sdk.NewSDK()
	if err != nil {
		log.Errorf("%s Could not connect to sdk: %v.\n", logger.LOG_Main, err)
	}

	roomChan := make(chan *game.Room)

	roomInit := false
	var matchmakerPodName string
	var dbMapID string
	var myGameServer *serverSDK.GameServer
	var playerIDs [setting.PLAYER_NUMBER]string
	agonesSDK.WatchGameServer(func(gs *serverSDK.GameServer) {
		// log.Infof("%s 遊戲房狀態 %s", logger.LOG_Main, gs.Status.State)
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%s 遊戲崩潰: %v.\n", logger.LOG_Main, err)
				shutdownServer(agonesSDK)
			}
		}()

		if !roomInit && gs.ObjectMeta.Labels["RoomName"] != "" {
			log.Infof("%s 開始初始化遊戲房!", logger.LOG_Main)
			matchmakerPodName = gs.ObjectMeta.Labels["MatchmakerPodName"]
			var pIDs [setting.PLAYER_NUMBER]string
			for i, v := range pIDs {
				key := fmt.Sprintf("Player%d", i)
				v = gs.ObjectMeta.Labels[key]
				playerIDs[i] = v
			}

			// 初始化MongoDB設定
			mongoAPIPublicKey := os.Getenv("MongoAPIPublicKey")
			mongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey")
			mongoUser := os.Getenv("MongoUser")
			mongoPW := os.Getenv("MongoPW")
			initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey, mongoUser, mongoPW)

			dbMapID = gs.ObjectMeta.Labels["DBMapID"]
			roomInit = true
			myGameServer = gs
			roomName := gs.ObjectMeta.Labels["RoomName"]
			podName := gs.ObjectMeta.Name

			nodeName := "test"
			matchmakerPodName := "test"
			log.Infof("%s ==============InitGameRoom==============", logger.LOG_Main)
			log.Infof("%s podName: %v", logger.LOG_Main, podName)
			log.Infof("%s nodeName: %v", logger.LOG_Main, nodeName)
			log.Infof("%s MatchmakerPodName: %s", logger.LOG_Main, matchmakerPodName)
			log.Infof("%s PlayerIDs: %s", logger.LOG_Main, playerIDs)
			log.Infof("%s dbMapID: %s", logger.LOG_Main, dbMapID)
			log.Infof("%s roomName: %s", logger.LOG_Main, roomName)
			log.Infof("%s Address: %s", logger.LOG_Main, myGameServer.Status.Address)
			log.Infof("%s Port: %v", logger.LOG_Main, myGameServer.Status.Ports[0].Port)
			log.Infof("%s ==============Info Finished==============", logger.LOG_Main)

			game.InitGameRoom(dbMapID, playerIDs, roomName, myGameServer.Status.Address, myGameServer.Status.Ports[0].Port, podName, nodeName, matchmakerPodName, roomChan)
			log.Infof("%s Init Game Room Success", logger.LOG_Main)
		} else {
			if matchmakerPodName != "" && gs.ObjectMeta.Labels["MatchmakerPodName"] != "" && matchmakerPodName != gs.ObjectMeta.Labels["MatchmakerPodName"] {
				log.Errorf("%s Agones has allocate error in parelle", logger.LOG_Main)

				// 要改成mongodb寫log版本
				// FirebaseFunction.WriteErrorLog(map[string]interface{}{
				// 	"ErrorID":    "ALLOCATE ERROR",
				// 	"Message":    "Agones has allocate error in parelle.",
				// 	"CreateTime": time.Now(),
				// })
			}
		}
	})

	// 將此遊戲房伺服器狀態標示為Ready(要標示為ready才會被Agones Allocation服務分配到)
	if err := agonesSDK.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
		return
	} else {
		log.Infof("%s Matchgame準備就緒 可被Agones Allocation服務分配", logger.LOG_Main)
	}

	stopChan := make(chan struct{})
	endGameChan := make(chan struct{})

	// Agones伺服器健康檢查
	go agonesHealthPin(agonesSDK, stopChan)

	// 等拿到房間資料後才開啟socket連線
	room := <-roomChan
	log.Infof("%s Got room data", logger.LOG_Main)
	close(roomChan)

	// 開啟連線
	log.Infof("%s Open TCP Connection", logger.LOG_Main)

	src := ":" + *port
	go openConnectTCP(agonesSDK, stopChan, src, room)
	go OpenConnectUDP(agonesSDK, stopChan, src, room)
	// 寫入DBMatchgame
	writeMatchgameToDB(*room.DBMatchgame)

	// 開始遊戲房主循環
	room.StartRun(stopChan, endGameChan)

	log.Infof("%s ==============MATCHGAME準備就緒==============", logger.LOG_Main)

	select {
	case <-stopChan:
		// 錯誤發生寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s game stop chan", logger.LOG_Main)
		shutdownServer(agonesSDK)
		return
	case <-endGameChan:
		// 遊戲房關閉寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s End game chan", logger.LOG_Main)
		delayShutdownServer(60*time.Second, agonesSDK, stopChan)
	}
	<-stopChan

	shutdownServer(agonesSDK) // 關閉Server
}
func writeMatchgameToDB(matchgame mongo.DBMatchgame) {
	log.Infof("%s 開始寫入Matchgame到DB", logger.LOG_Main)
	_, err := mongo.AddDocByStruct(mongo.ColName.Matchgame, matchgame)
	if err != nil {
		log.Errorf("%s writeMatchgameToDB: %v", logger.LOG_Main, err)
		return
	}
	log.Infof("%s 寫入Matchgame到DB完成", logger.LOG_Main)
}

// 初始化MongoDB設定
func initMonogo(mongoAPIPublicKey string, mongoAPIPrivateKey string, user string, pw string) {
	log.Infof("%s 初始化mongo開始", logger.LOG_Main)
	mongo.Init(mongo.InitData{
		Env:           Env,
		APIPublicKey:  mongoAPIPublicKey,
		APIPrivateKey: mongoAPIPrivateKey,
	}, user, pw)
	log.Infof("%s 初始化mongo完成", logger.LOG_Main)
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
func openConnectTCP(s *sdk.SDK, stop chan struct{}, src string, room *game.Room) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectTCP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	tcpListener, err := net.Listen("tcp", src)
	if err != nil {
		log.Errorf("%s Listen error: %v.\n", logger.LOG_Main, err)
	}
	defer tcpListener.Close()
	log.Infof("%s TCP server start and listening on %s", logger.LOG_Main, src)

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Errorf("%s Unable to accept incoming tcp connection: %v.\n", logger.LOG_Main, err)
			continue
		}
		go handleConnectionTCP(conn, stop, room)
	}
}

// 開啟UDP連線
func OpenConnectUDP(s *sdk.SDK, stop chan struct{}, src string, room *game.Room) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectUDP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	conn, err := net.ListenPacket("udp", src)
	if err != nil {
		log.Errorf("%s Could not start udp server: %v.\n", logger.LOG_Main, err)
	}
	defer conn.Close()
	log.Infof("%s UDP server start and listening on %s", logger.LOG_Main, src)

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
		for _, t := range connnTokens {
			log.Infof("%s Connection Tokens : %s", logger.LOG_Main, t)
			if t == txt {
				hasToken = true
			}
		}
		if hasToken {
			// log.Infof("%s Start Update UDP Message", logger.LOG_Main)
			go handleConnectionUDP(conn, stop, sender, room)
		}
	}
}

// 處理TCP連線封包，目前只處理加房驗證，之後遊戲內通訊改由UDP處理
func handleConnectionTCP(conn net.Conn, stop chan struct{}, room *game.Room) {
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
			room.PlayerLeave(conn)
			return
		}
		log.Infof("%s Receive: %s from %s \n", logger.LOG_Main, pack.CMD, remoteAddr)

		//未驗證前，除了Auth指令進來其他都擋掉
		if !isAuth && pack.CMD != packet.AUTH {
			log.Infof("%s UnAuth command", logger.LOG_Main)
			return
		}
		log.Infof("%s pack.CMD: %s", logger.LOG_Main, pack.CMD)
		if pack.CMD == packet.AUTH {

			authContent := packet.AuthCMD{}
			if ok := authContent.Parse(pack.Content); !ok {
				log.Errorf("%s Parse AuthCMD Failed", logger.LOG_Main)
				return
			}

			// 驗證Token是否合法
			// token, err := FirebaseFunction.VerifyIDToken(authContent.Token)
			if err != nil {
				log.Errorf("%s Verify ID token failed: %v\n", logger.LOG_Main, err)
				_ = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_REPLY,
					PackID: pack.PackID,
					ErrMsg: err.Error(),
					Content: &packet.AuthCMD_Reply{
						IsAuth: false,
					},
				})
				return
			}

			// 像mongodb atlas驗證token並取得playerID 有通過驗證後才處理後續
			playerID, authErr := mongo.PlayerVerify(authContent.Token)
			// 驗證失敗
			if authErr != nil || playerID == "" {
				log.Errorf("%s Player verify failed: %v", logger.LOG_Main, authErr)
				_ = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_REPLY,
					PackID: pack.PackID,
					ErrMsg: "Auth toekn驗證失敗",
					Content: &packet.AuthCMD_Reply{
						IsAuth: false,
					},
				})
			}

			// 通過驗證後回送驗證結果與連線toekn
			isAuth = true
			newConnToken := generateSecureToken(32)
			err = packet.SendPack(encoder, &packet.Pack{
				CMD:    packet.AUTH_REPLY,
				PackID: pack.PackID,
				Content: &packet.AuthCMD_Reply{
					IsAuth:    true,
					ConnToken: newConnToken,
				},
			})
			if err != nil {
				return
			}
			defer removeConnectionToken(newConnToken)
			connnTokens = append(connnTokens, newConnToken)

			// 將玩家加入遊戲房
			player := game.Player{
				ID: playerID,
				ConnTCP: game.ConnectionTCP{
					Conn:    conn,
					Encoder: encoder,
					Decoder: decoder,
				},
			}
			room.PlayerJoin(player)
		} else {
			err = room.HandleMessage(conn, pack, stop)
			if err != nil {
				log.Errorf("%s GameRoom Handle Message Error: %v\n", logger.LOG_Main, err.Error())
				room.PlayerLeave(conn)
				return
			}
		}
	}
}

// 處理UDP連線封包
func handleConnectionUDP(conn net.PacketConn, stop chan struct{}, addr net.Addr, room *game.Room) {
	timer := time.NewTicker(matchgameSetting.TIME_UPDATE_INTERVAL_MS * time.Millisecond)
	for {
		select {
		case <-stop:
			//被強制終止
			return
		case <-timer.C:
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATE_UDP,
				PackID: -1,
				Content: game.ServerStateContent{
					ServerTime: room.PassSecs,
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
func agonesHealthPin(agonesSDK *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(matchgameSetting.AGONES_HEALTH_PIN_INTERVAL_SEC * time.Second)
	for {
		if err := agonesSDK.Health(); err != nil {
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
	for i, v := range connnTokens {
		if v == token {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	after := append(connnTokens[:index], connnTokens[index+1:]...)
	connnTokens = after
}
