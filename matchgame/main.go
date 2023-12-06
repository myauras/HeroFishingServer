package main

import (
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	gSetting "matchgame/setting"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"

	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"matchgame/game"
	"matchgame/packet"
	"net"
	"os"
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

var Env string // 環境版本

func main() {
	// 設定日誌級別
	log.SetLevel(log.InfoLevel)
	// 設定日誌輸出，預設為標準輸出
	log.SetOutput(os.Stdout)
	// 自定義時間格式，包含毫秒
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Main Crash: %v", r)
		}
	}()

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
	InitGameJson() // 初始化遊戲Json資料

	// 初始化redisDB
	log.Infof("%s 開始初始化RedisDB", logger.LOG_Main)
	redis.Init()
	redisErr := redis.Ping()
	if redisErr != nil {
		log.Errorf("%s 初始化RedisDB發生錯誤: %v", logger.LOG_Main, redisErr)
	} else {
		log.Infof("%s 初始化RedisDB完成", logger.LOG_Main)
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
			nodeName := os.Getenv("NodeName")
			log.Infof("%s ==============InitGameRoom==============", logger.LOG_Main)
			log.Infof("%s podName: %v", logger.LOG_Main, podName)
			log.Infof("%s nodeName: %v", logger.LOG_Main, nodeName)
			log.Infof("%s PlayerIDs: %s", logger.LOG_Main, playerIDs)
			log.Infof("%s dbMapID: %s", logger.LOG_Main, dbMapID)
			log.Infof("%s roomName: %s", logger.LOG_Main, roomName)
			log.Infof("%s Address: %s", logger.LOG_Main, myGameServer.Status.Address)
			log.Infof("%s Port: %v", logger.LOG_Main, myGameServer.Status.Ports[0].Port)
			log.Infof("%s Get Info Finished", logger.LOG_Main)

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

	src := ":" + *port
	go openConnectTCP(agonesSDK, stopChan, src)
	go openConnectUDP(agonesSDK, stopChan, src)
	// 寫入DBMatchgame
	writeMatchgameToDB(*room.DBMatchgame)

	// 開始遊戲房計時器
	go room.RoomTimer(stopChan)

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

// 初始化遊戲Json資料
func InitGameJson() {
	log.Infof("%s 開始初始化GameJson", logger.LOG_Main)
	err := gameJson.Init(Env)
	if err != nil {
		log.Errorf("%s 初始化GameJson失敗: %v", logger.LOG_Main, err)
		return
	}
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
func openConnectTCP(s *sdk.SDK, stop chan struct{}, src string) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectTCP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	tcpListener, err := net.Listen("tcp", src)
	if err != nil {
		log.Errorf("%s (TCP)偵聽失敗: %v.\n", logger.LOG_Main, err)
	}
	defer tcpListener.Close()
	log.Infof("%s (TCP)開始偵聽 %s", logger.LOG_Main, src)

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Errorf("%s Unable to accept incoming tcp connection: %v.\n", logger.LOG_Main, err)
			continue
		}
		go handleConnectionTCP(conn, stop)
	}
}

// 開啟UDP連線
func openConnectUDP(s *sdk.SDK, stop chan struct{}, src string) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectUDP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	conn, err := net.ListenPacket("udp", src)
	if err != nil {
		log.Errorf("%s (UDP)偵聽失敗: %v.\n", logger.LOG_Main, err)
	}
	defer conn.Close()
	log.Infof("%s (UDP)開始偵聽 %s", logger.LOG_Main, src)

	for {
		// 取得收到的封包
		buffer := make([]byte, 1024)
		n, addr, readBufferErr := conn.ReadFrom(buffer)
		if readBufferErr != nil {
			log.Errorf("%s (UDP)讀取封包錯誤: %v", logger.LOG_Main, readBufferErr)
			continue
		}
		if n <= 0 {
			continue
		}
		// 解析json數據
		var pack packet.UDPReceivePack
		unmarshalErr := json.Unmarshal(buffer[:n], &pack)
		if unmarshalErr != nil {
			log.Errorf("%s (UDP)解析封包錯誤: %s", logger.LOG_Main, unmarshalErr.Error())
			continue
		}
		// 玩家驗證
		player := game.MyRoom.GetPlayerByConnToken(pack.ConnToken)

		if player == nil {
			log.Errorf("%s (UDP)Token驗證失敗 來自 %s 的命令: %s \n", logger.LOG_Main, addr.String(), pack.CMD)
			continue
		}
		log.Infof("%s (UDP)收到來自 %s 的命令: %s \n", logger.LOG_Main, addr.String(), pack.CMD)

		// 執行命令
		if pack.CMD == packet.UDPAUTH {
			if player.ConnUDP.Conn != nil {
				log.Errorf("%s (UDP)此玩家已執行過UDP Auth有正在進行的updateGameLoop", logger.LOG_Main)
				continue
			}
			// 更新連線資料
			player.ConnUDP.Conn = conn
			player.ConnUDP.Addr = addr
			go updateGameLoop(player, stop)
		} else {
			if player.ConnUDP.Conn == nil || player.ConnUDP.Addr == nil {
				log.Errorf("%s (UDP)收到來自 %s(%s) 但尚未進行UDP Auth的命令: %s", logger.LOG_Main, player.DBPlayer.ID, addr, pack.CMD)
			}
			// 更新連線資料
			player.ConnUDP.Conn = conn
			if player.ConnUDP.Addr.String() != addr.String() { // 玩家通過ConnToken驗證但Addr有變更可能是因為Wifi環境改變
				log.Infof("%s (UDP)玩家 %s 的位置從 %s 變更為 %s \n", logger.LOG_Main, player.DBPlayer.ID, player.ConnUDP.Addr.String(), addr.String())
				// 更新address避免客戶端的網路位置有改變這樣對於Wifi變更的用戶體驗比較好
				// 但是要注意若之後有使用udp送重要行為 為了避免connToken被封包攔截要讓玩家需要重新通過tcp auth取新的token才是安全的作法
				player.ConnUDP.Addr = addr
			}
			handleConnectionUDP(player, pack, stop)
		}
	}
}

// 處理TCP連線封包
func handleConnectionTCP(conn net.Conn, stop chan struct{}) {
	remoteAddr := conn.RemoteAddr().String()
	// log.Infof("%s Client %s connected", logger.LOG_Main, conn.RemoteAddr().String())
	defer conn.Close()
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s (TCP)處理封包錯誤: %v.", logger.LOG_Main, err)
		}
	}()
	isAuth := false
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(1 * time.Minute))
	for {
		select {
		case <-stop:
			log.Errorf("強制終止TCP")
			// 被強制終止
			return
		default:
		}
		pack, err := packet.ReadPack(decoder)
		if err != nil {
			game.MyRoom.KickPlayer(true, conn)
			return
		}
		log.Infof("%s (TCP)收到來自 %s 的命令: %s \n", logger.LOG_Main, remoteAddr, pack.CMD)

		//未驗證前，除了Auth指令進來其他都擋掉
		if !isAuth && pack.CMD != packet.AUTH {
			log.Infof("%s 收到未驗證的封包", logger.LOG_Main)
			return
		}
		if pack.CMD == packet.AUTH {
			authContent := packet.Auth{}
			if ok := authContent.Parse(pack.Content); !ok {
				log.Errorf("%s 反序列化AUTH封包失敗", logger.LOG_Main)
				return
			}
			// 像mongodb atlas驗證token並取得playerID 有通過驗證後才處理後續
			playerID, authErr := mongo.PlayerVerify(authContent.Token)
			// 驗證失敗
			if authErr != nil || playerID == "" {
				log.Errorf("%s 玩家驗證錯誤: %v", logger.LOG_Main, authErr)
				_ = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_TOCLIENT,
					PackID: pack.PackID,
					ErrMsg: "玩家驗證錯誤",
					Content: &packet.Auth_ToClient{
						IsAuth: false,
					},
				})
			}
			var dbPlayer mongo.DBPlayer
			getPlayerDocErr := mongo.GetDocByID(mongo.ColName.Player, playerID, &dbPlayer)
			if getPlayerDocErr != nil {
				log.Errorf("%s DBPlayer資料錯誤: %v", logger.LOG_Main, getPlayerDocErr)
				_ = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_TOCLIENT,
					PackID: pack.PackID,
					ErrMsg: "DBPlayer資料錯誤",
					Content: &packet.Auth_ToClient{
						IsAuth: false,
					},
				})
			}

			isAuth = true
			// 建立RedisDB Player
			redisPlayer, redisPlayerErr := redis.CreatePlayerData(dbPlayer.ID, int(dbPlayer.Point), int(dbPlayer.HeroExp))
			if redisPlayerErr != nil {
				log.Errorf("%s 建立RedisPlayer錯誤: %v", logger.LOG_Main, redisPlayerErr)
				_ = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_TOCLIENT,
					PackID: pack.PackID,
					ErrMsg: "建立RedisPlayer錯誤",
					Content: &packet.Auth_ToClient{
						IsAuth: false,
					},
				})
			}
			redisPlayer.StartInGameUpdatePlayer() // 開始跑玩家資料定時更新上RedisDB程序

			// 將該玩家monogoDB上的redisSync設為false
			updatePlayerBson := bson.D{
				{Key: "redisSync", Value: false},
			}
			mongo.UpdateDocByID(mongo.ColName.Player, dbPlayer.ID, updatePlayerBson)

			// 建立udp socket連線Token
			newConnToken := generateSecureToken(32)

			// 將玩家加入遊戲房
			player := gSetting.Player{
				DBPlayer:     &dbPlayer,
				RedisPlayer:  redisPlayer,
				LastUpdateAt: time.Now(),
				ConnTCP: &gSetting.ConnectionTCP{
					Conn:    conn,
					Encoder: encoder,
					Decoder: decoder,
				},
				ConnUDP: &gSetting.ConnectionUDP{
					ConnToken: newConnToken,
				},
			}
			joined := game.MyRoom.JoinPlayer(&player)
			if !joined {
				log.Errorf("%s 玩家加入房間失敗", logger.LOG_Main)
				return
			}
			// 回送client
			err = packet.SendPack(encoder, &packet.Pack{
				CMD:    packet.AUTH_TOCLIENT,
				PackID: pack.PackID,
				Content: &packet.Auth_ToClient{
					IsAuth:    true,
					ConnToken: newConnToken,
					Index:     player.Index,
				},
			})
			if err != nil {
				return
			}

		} else {
			err = game.MyRoom.HandleMessage(conn, pack, stop)
			if err != nil {
				log.Errorf("%s (TCP)處理GameRoom封包錯誤: %v\n", logger.LOG_Main, err.Error())
				game.MyRoom.KickPlayer(true, conn)
				return
			}
		}
	}
}

// 處理UDP連線封包
func handleConnectionUDP(player *gSetting.Player, pack packet.UDPReceivePack, stop chan struct{}) {
	switch {
	case pack.CMD == packet.UPDATEGAME:
		// log.Infof("%s 更新玩家 %s 心跳", logger.LOG_Main, player.DBPlayer.ID)
		player.LastUpdateAt = time.Now() // 更新心跳
	}
}

// 定時更新遊戲狀態給Client
func updateGameLoop(player *gSetting.Player, stop chan struct{}) {
	log.Infof("%s (UDP)開始updateGameLoop", logger.LOG_Main)
	timer := time.NewTicker(gSetting.TIME_UPDATE_INTERVAL_MS * time.Millisecond)
	for {
		select {
		case <-stop:
			//被強制終止
			log.Errorf("強制終止UDP")
			return
		case <-timer.C:
			if player == nil || player.ConnUDP.Conn == nil {
				return
			}
			// 定時送遊戲更新給Client
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATEGAME_TOCLIENT,
				PackID: -1,
				Content: &packet.UpdateGame_ToClient{
					GameTime: game.MyRoom.GameTime,
				},
			})
			if err != nil {
				log.Errorf("%s (UDP)序列化UPDATEGAME封包錯誤. %s", logger.LOG_Main, err.Error())
				continue
			}
			sendData = append(sendData, '\n')
			_, sendErr := player.ConnUDP.Conn.WriteTo(sendData, player.ConnUDP.Addr)
			if sendErr != nil {
				log.Errorf("%s (UDP)送UPDATEGAME封包錯誤 %s", logger.LOG_Main, sendErr.Error())
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
	tick := time.Tick(gSetting.AGONES_HEALTH_PIN_INTERVAL_SEC * time.Second)
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
