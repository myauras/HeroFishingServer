package main

import (
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	// gSetting "matchgame/setting"

	log "github.com/sirupsen/logrus"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	serverSDK "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	"flag"
	// "fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"matchgame/agones"
	"matchgame/game"
	"os"
	"time"
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

	if imgVer := os.Getenv("ImgVer"); imgVer != "" {
		log.Infof("%s Image版本為: %s", logger.LOG_Main, imgVer)
	} else {
		log.Errorf("%s 取不到環境變數: ImgVer", logger.LOG_Main)
	}
	Env = *flag.String("Version", "Dev", "version setting")
	if ep := os.Getenv("Version"); ep != "" {
		Env = ep
	}
	err := agones.InitAgones()
	if err != nil {
		log.Errorf("%s %s", logger.LOG_Main, err)
	}
	InitGameJson() // 初始化遊戲Json資料

	roomChan := make(chan *game.Room)
	roomInit := false
	var matchmakerPodName string
	var dbMapID string
	var myGameServer *serverSDK.GameServer

	agones.AgonesSDK.WatchGameServer(func(gs *serverSDK.GameServer) {
		// log.Infof("%s 遊戲房狀態 %s", logger.LOG_Main, gs.Status.State)
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%s 遊戲崩潰: %v.\n", logger.LOG_Main, err)
				agones.AgonesSDK.Shutdown()
			}
		}()

		if !roomInit && gs.ObjectMeta.Labels["RoomName"] != "" {
			log.Infof("%s 開始初始化遊戲房!", logger.LOG_Main)

			matchmakerPodName = gs.ObjectMeta.Labels["MatchmakerPodName"]

			playerIDs := [setting.PLAYER_NUMBER]string{}
			// 這裡先不把玩家加到DB中(因為之後會透過JoinPlayer來加)
			// for i := range playerIDs {
			// 	key := fmt.Sprintf("Player%d", i)
			// 	playerIDs[i] = gs.ObjectMeta.Labels[key]
			// }

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
			log.Infof("%s ==============第一位玩家加入 開始初始化房間==============", logger.LOG_Main)
			log.Infof("%s podName: %v", logger.LOG_Main, podName)
			log.Infof("%s nodeName: %v", logger.LOG_Main, nodeName)
			log.Infof("%s PlayerIDs: %s", logger.LOG_Main, playerIDs)
			log.Infof("%s dbMapID: %s", logger.LOG_Main, dbMapID)
			log.Infof("%s roomName: %s", logger.LOG_Main, roomName)
			log.Infof("%s Address: %s", logger.LOG_Main, myGameServer.Status.Address)
			log.Infof("%s Port: %v", logger.LOG_Main, myGameServer.Status.Ports[0].Port)
			log.Infof("%s Get Info Finished", logger.LOG_Main)

			game.InitGameRoom(dbMapID, playerIDs, roomName, myGameServer.Status.Address, myGameServer.Status.Ports[0].Port, podName, nodeName, matchmakerPodName, roomChan)

			log.Infof("%s GameServer狀態為: %s", logger.LOG_Main, gs.Status.State)
			log.Infof("%s ==============初始化房間完成==============", logger.LOG_Main)
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

	// go TestLoop() // 測試Loop

	stopChan := make(chan struct{})
	endGameChan := make(chan struct{})
	agones.SetServerState(agonesv1.GameServerStateReady) // 設定房間為Ready(才有人能加進來)
	// Agones伺服器健康檢查
	go agones.AgonesHealthPin(stopChan)

	// 等拿到房間資料後才開啟socket連線
	room := <-roomChan

	close(roomChan)

	// 初始化redisDB
	redis.Init()

	// 開啟連線
	src := ":" + *port
	go openConnectTCP(agones.AgonesSDK, stopChan, src)
	go openConnectUDP(agones.AgonesSDK, stopChan, src)
	// 寫入DBMatchgame(加入已存在房間時, DBMatchgame的玩家加入是在Matchmaker寫入, 但開房是在DBMatchgame寫入)
	room.WriteMatchgameToDB()
	// 開始遊戲房計時器
	go room.RoomTimer(stopChan)
	// 開始生怪
	go room.MSpawner.SpawnTimer()
	room.MSpawner.SpawnSwitch(true)

	log.Infof("%s ==============房間準備就緒==============", logger.LOG_Main)

	select {
	case <-stopChan:
		// 錯誤發生寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s game stop chan", logger.LOG_Main)
		agones.ShutdownServer()
		return
	case <-endGameChan:
		// 遊戲房關閉寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s End game chan", logger.LOG_Main)
		agones.DelayShutdownServer(60*time.Second, stopChan)
	}
	<-stopChan

	agones.ShutdownServer() // 關閉Server
}

// 房間循環
func TestLoop() {
	if agones.AgonesSDK == nil {
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {

		gs, err := agones.AgonesSDK.GameServer()
		if err != nil {
			log.Fatalf("取得GameServer失敗: %v", err)
		}
		log.Infof("%s GameServer狀態為: %s", logger.LOG_Main, gs.Status.State)
	}
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
