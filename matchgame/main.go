package main

import (
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	gSetting "matchgame/setting"

	log "github.com/sirupsen/logrus"

	"flag"
	"fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"matchgame/game"
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

	log.Infof("%s ==============MATCHGAME 啟動3==============", logger.LOG_Main)
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

	// 將此遊戲房伺服器狀態標示為Ready(要標示為ready才會被Agones Allocation服務分配到)
	if err := agonesSDK.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
		return
	} else {
		log.Infof("%s Matchgame已可被Agones Allocation服務分配", logger.LOG_Main)
	}

	stopChan := make(chan struct{})
	endGameChan := make(chan struct{})

	// Agones伺服器健康檢查
	go agonesHealthPin(agonesSDK, stopChan)

	// 等拿到房間資料後才開啟socket連線
	room := <-roomChan

	close(roomChan)

	// 初始化redisDB
	redis.Init()

	// 開啟連線
	src := ":" + *port
	go openConnectTCP(agonesSDK, stopChan, src)
	go openConnectUDP(agonesSDK, stopChan, src)
	// 寫入DBMatchgame
	writeMatchgameToDB(*room.DBMatchgame)
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

// 送定時送Agones健康ping通知agones server遊戲房還活著
// Agones的超時為periodSeconds設定的秒數 參考官方: https://agones.dev/site/docs/guides/health-checking/
func agonesHealthPin(agonesSDK *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(gSetting.AGONES_HEALTH_PIN_INTERVAL_SEC * time.Second)
	for {
		if err := agonesSDK.Health(); err != nil {
			log.Errorf("%s ping agones server錯誤: %v", logger.LOG_Main, err)
		}
		select {
		case <-stop:
			log.Infof("%s Health pings 意外停止", logger.LOG_Main)
			return
		case <-tick:
		}
	}
}
