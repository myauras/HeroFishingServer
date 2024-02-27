package main

import (
	"herofishingGoModule/k8s"
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	"strconv"

	// gSetting "matchgame/setting"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"

	"flag"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	serverSDK "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"

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

var PodName string // Pod名稱

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
	port := flag.String("port", "7654", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}

	if imgVer := os.Getenv("ImgVer"); imgVer != "" {
		log.Infof("%s Image版本為: %s", logger.LOG_Main, imgVer)
	} else {
		log.Errorf("%s 取不到環境變數: ImgVer", logger.LOG_Main)
	}
	if game.Mode = os.Getenv("Mode"); game.Mode != "" {
		log.Infof("%s Mode為: %s", logger.LOG_Main, game.Mode)
	} else {
		game.Mode = "standard"
		log.Errorf("%s 取不到環境變數: Mode", logger.LOG_Main)
	}
	if PodName = os.Getenv("PodName"); PodName != "" {
		log.Infof("%s PodName為: %s", logger.LOG_Main, PodName)
	} else {
		log.Errorf("%s 取不到環境變數: PodName", logger.LOG_Main)
	}
	Env = *flag.String("Version", "Dev", "version setting")
	if ep := os.Getenv("Version"); ep != "" {
		Env = ep
	}
	if game.Mode != "non-agones" { // non-agones模式下不要呼叫初始化Agones, 也不要偵聽agones訊號
		err := agones.InitAgones()
		if err != nil {
			log.Errorf("%s %s", logger.LOG_Main, err)
		}
		go signalListen()
	}
	InitGameJson() // 初始化遊戲Json資料
	roomChan := make(chan *game.Room)
	var packID = int64(0)

	if game.Mode == "standard" { // standard模式
		var matchmakerPodName string
		var err error
		roomInit := false

		agones.AgonesSDK.WatchGameServer(func(gs *serverSDK.GameServer) {

			// log.Infof("%s 遊戲房狀態 %s", logger.LOG_Main, gs.Status.State)
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("%s 遊戲崩潰: %v.\n", logger.LOG_Main, err)
					agones.AgonesSDK.Shutdown()
				}
			}()

			if !roomInit && gs.ObjectMeta.Labels["RoomName"] != "" {
				log.Infof("%s 開始房間建立", logger.LOG_Main)

				matchmakerPodName = gs.ObjectMeta.Labels["MatchmakerPodName"]

				playerIDs := [setting.PLAYER_NUMBER]string{}

				// 初始化MongoDB設定
				mongoAPIPublicKey := os.Getenv("MongoAPIPublicKey")
				mongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey")
				mongoUser := os.Getenv("MongoUser")
				mongoPW := os.Getenv("MongoPW")
				initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey, mongoUser, mongoPW)

				dbMapID := gs.ObjectMeta.Labels["DBMapID"]
				roomInit = true
				packID, err = strconv.ParseInt(gs.ObjectMeta.Labels["PackID"], 10, 64)
				if err != nil {
					log.Errorf("%s strconv.ParseInt packID錯誤: %v", logger.LOG_Main, err)
				}
				roomName := gs.ObjectMeta.Labels["RoomName"]
				podName := gs.ObjectMeta.Name
				nodeName := os.Getenv("NodeName")
				log.Infof("%s ==============開始初始化房間==============", logger.LOG_Main)
				log.Infof("%s packID: %v", logger.LOG_Main, packID)
				log.Infof("%s podName: %v", logger.LOG_Main, podName)
				log.Infof("%s nodeName: %v", logger.LOG_Main, nodeName)
				log.Infof("%s PlayerIDs: %s", logger.LOG_Main, playerIDs)
				log.Infof("%s dbMapID: %s", logger.LOG_Main, dbMapID)
				log.Infof("%s roomName: %s", logger.LOG_Main, roomName)
				log.Infof("%s Address: %s", logger.LOG_Main, gs.Status.Address)
				log.Infof("%s Port: %v", logger.LOG_Main, gs.Status.Ports[0].Port)
				log.Infof("%s Get Info Finished", logger.LOG_Main)

				game.InitGameRoom(dbMapID, playerIDs, roomName, gs.Status.Address, gs.Status.Ports[0].Port, podName, nodeName, matchmakerPodName, roomChan)
				agones.SetServerState(agonesv1.GameServerStateAllocated) // 設定房間為Allocated(agones應該會在WatchGameServer後自動設定為Allocated但這邊還是主動設定)
				log.Infof("%s GameServer狀態為: %s", logger.LOG_Main, gs.Status.State)
				log.Infof("%s ==============初始化房間完成==============", logger.LOG_Main)
			} else {
				if matchmakerPodName != "" && gs.ObjectMeta.Labels["MatchmakerPodName"] != "" && matchmakerPodName != gs.ObjectMeta.Labels["MatchmakerPodName"] {
					log.Errorf("%s Agones has allocate error in parelle", logger.LOG_Main)
				}
			}
		})
	} else if game.Mode == "non-agones" { // non-agones模式

		log.Infof("%s 開始房間建立", logger.LOG_Main)

		go func() {

			myPort, parsePortErr := strconv.ParseInt(*port, 10, 32)
			if parsePortErr != nil {
				log.Errorf("%s parse Port錯誤: %v", logger.LOG_Main, parsePortErr)
			}

			// 初始化MongoDB設定
			mongoAPIPublicKey := os.Getenv("MongoAPIPublicKey")
			mongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey")
			mongoUser := os.Getenv("MongoUser")
			mongoPW := os.Getenv("MongoPW")
			initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey, mongoUser, mongoPW)

			// 取Loadbalancer分配給此pod的對外IP並寫入資料庫(non-agones的連線方式不會透過Matchmaker分配房間再把ip回傳給client, 而是直接讓client去連資料庫matchgame的ip)
			// 因為每個LoadBalancer Service似乎不支持
			log.Infof("%s 取Loadbalancer分配給此pod的對外IP.\n", logger.LOG_Main)
			tcpIP := ""
			udpIP := ""
			for tcpIP == "" && udpIP == "" {
				// 因為pod啟動後Loadbalancer並不會立刻就分配好ip(會有延遲) 所以每5秒取一次 直到取到ip才往下跑
				time.Sleep(5 * time.Second) // 每5秒取一次ip
				// 取TCP服務開放的對外IP
				if tcpIP == "" {
					getTcpIP, getTcpIPErr := getExternalIP(setting.MATCHGAME_TESTVER_TCP)
					if getTcpIPErr != nil {
						// 取得ip失敗
						break
					}
					if getTcpIP != "" {
						tcpIP = getTcpIP
						log.Infof("%s 取得對外TCP IP成功: %s .\n", logger.LOG_Main, tcpIP)
					}
				}

				// 取UDP服務開放的對外IP
				if udpIP == "" {
					getUdpIP, getUdpIPErr := getExternalIP(setting.MATCHGAME_TESTVER_UDP)
					if getUdpIPErr != nil {
						// 取得ip失敗
						break
					}
					if getUdpIP != "" {
						udpIP = getUdpIP
						log.Infof("%s 取得對外UDP IP成功: %s .\n", logger.LOG_Main, udpIP)
					}
				}

			}
			setExternalIPandPort(tcpIP, udpIP, int32(myPort))
			matchmakerPodName := ""
			playerIDs := [setting.PLAYER_NUMBER]string{}

			// 依據DBGameSetting中取GameState設定
			log.Infof("%s 取DBGameState資料", logger.LOG_Main)
			var dbGameState mongo.DBGameState
			dbGameStateErr := mongo.GetDocByID(mongo.ColName.GameSetting, "GameState", &dbGameState)
			if dbGameStateErr != nil {
				log.Errorf("%s InitGameRoom時取DBGameState資料發生錯誤: %v", logger.LOG_Main, dbGameStateErr)
			}
			log.Infof("%s 取DBGameState資料成功", logger.LOG_Main)

			dbMapID := dbGameState.MatchgameTestverMapID
			roomName := dbGameState.MatchgameTestverRoomName
			nodeName := os.Getenv("NodeName")
			log.Infof("%s ==============開始初始化房間==============", logger.LOG_Main)
			log.Infof("%s podName: %v", logger.LOG_Main, PodName)
			log.Infof("%s nodeName: %v", logger.LOG_Main, nodeName)
			log.Infof("%s PlayerIDs: %s", logger.LOG_Main, playerIDs)
			log.Infof("%s dbMapID: %s", logger.LOG_Main, dbMapID)
			log.Infof("%s roomName: %s", logger.LOG_Main, roomName)
			log.Infof("%s Port: %v", logger.LOG_Main, port)
			log.Infof("%s Get Info Finished", logger.LOG_Main)

			game.InitGameRoom(dbMapID, playerIDs, roomName, "", int32(myPort), PodName, nodeName, matchmakerPodName, roomChan)

			log.Infof("%s ==============初始化房間完成==============", logger.LOG_Main)
		}()
	}

	// go TestLoop() // 測試Loop

	stopChan := make(chan struct{})
	endGameChan := make(chan struct{})
	if game.Mode != "non-agones" {
		agones.SetServerState(agonesv1.GameServerStateReady) // 設定房間為Ready(才會被Matchmaker分配玩家進來)
		go agones.AgonesHealthPin(stopChan)                  // Agones伺服器健康檢查
	}
	room := <-roomChan
	close(roomChan)
	// ====================Room資料設定完成====================
	log.Infof("%s ==============Room資料設定完成==============", logger.LOG_Main)
	redis.Init()              // 初始化redisDB
	room.WriteMatchgameToDB() // 寫入DBMatchgame(加入已存在房間時, DBMatchgame的玩家加入是在Matchmaker寫入, 但開房是在DBMatchgame寫入)

	// 開啟連線
	src := ":" + *port
	go openConnectTCP(stopChan, src)
	go openConnectUDP(stopChan, src)

	go room.RoomTimer(stopChan) // 開始遊戲房計時器

	// 開始生怪計時器
	go room.MSpawner.SpawnTimer()
	room.MSpawner.SpawnSwitch(false)
	if game.Mode != "non-agones" { // non-agones模式下不需與Matchmaker溝通
		room.PubGameCreatedMsg(int(packID)) // 送房間建立訊息給Matchmaker
		go room.SubMatchmakerMsg()          // 訂閱MatchmakerMsg
	}

	select {
	case <-stopChan:
		// 錯誤發生寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s game stop chan", logger.LOG_Main)
		if game.Mode != "non-agones" { // non-agones模式下不使用agones服務
			agones.ShutdownServer()
		}

		return
	case <-endGameChan:
		// 遊戲房關閉寫入Log
		// FirebaseFunction.DeleteGameRoom(RoomName)
		log.Infof("%s End game chan", logger.LOG_Main)
		if game.Mode != "non-agones" { // non-agones模式下不使用agones服務
			agones.DelayShutdownServer(60*time.Second, stopChan)
		}
	}
	<-stopChan

	if game.Mode != "non-agones" { // non-agones模式下不使用agones服務
		agones.ShutdownServer() // 關閉Server
	}

}

// 取Loadbalancer分配給此pod的對外
func getExternalIP(_serviceName string) (string, error) {
	ip, err := k8s.GetLoadBalancerExternalIP(setting.NAMESPACE_GAMESERVER, _serviceName)
	if err != nil {
		log.Errorf("%s GetLoadBalancerExternalIP error: %v.\n", logger.LOG_Main, err)
	}
	return ip, err
}

// 寫入對外ID到DB中
func setExternalIPandPort(tcpIP string, udpIP string, port int32) {
	log.Infof("%s 開始寫入對外IP到DB tcpIP:%s udpIP:%s port:%v.\n", logger.LOG_Main, tcpIP, udpIP, port)
	// 設定要更新的資料
	data := bson.D{
		{Key: "matchgame-testver-tcp-ip", Value: tcpIP},
		{Key: "matchgame-testver-udp-ip", Value: udpIP},
		{Key: "matchgame-testver-port", Value: port},
	}
	// 更新資料
	_, err := mongo.UpdateDocByBsonD(mongo.ColName.GameSetting, "GameState", data)
	if err != nil {
		log.Errorf("%s SetExternalID失敗: %v", logger.LOG_Main, err)
		return
	}
	log.Infof("%s 寫入對外IP完成.\n", logger.LOG_Main)
}

// 房間循環
func TestLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {

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
