package main

import (
	logger "crontasker/logger"
	"flag"
	"fmt"
	mongo "herofishingGoModule/mongo"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

// Cron格式參考: https://crontab.cronhub.io/
const (
	PLAYER_OFFLINE_CRON              = "*/3 * * * *" // 離線檢測時間
	PLAYER_OFFLINE_THRESHOLD_MINUTES = 3             // 上次更新離現在超過X分鐘算是離線
)

var Env string // 環境版本

func main() {
	log.Infof("%s ==============MATCHGAME 啟動==============", logger.LOG_Main)

	// 設定環境版本
	Env = *flag.String("Env", "Dev", "Env setting")
	if envEnv := os.Getenv("Env"); envEnv != "" {
		Env = envEnv
	}
	log.Infof("%s Env: %s", logger.LOG_Main, Env)

	// 初始化MongoDB設定
	mongoAPIPublicKey := os.Getenv("MongoAPIPublicKey")
	mongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey")
	mongoUser := os.Getenv("MongoUser")
	mongoPW := os.Getenv("MongoPW")
	initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey, mongoUser, mongoPW)

	myCron := cron.New()
	_, err := myCron.AddFunc(PLAYER_OFFLINE_CRON, playerOfflineHandler)
	if err != nil {
		log.Infof("%s 安排playerOfflineHandler錯誤: %v \n", logger.LOG_Main, err)
		return
	}
	myCron.Start()

	select {}
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

// 使用playerIds清單找出PlayerState表中 _id為playerIDs中且lastUpdatedAt欄位的時間小於minutesBefore的表
// 如果有表符合以上條件就把對應_id的Player表的onlineState改為Offline
func playerOfflineHandler() {
	log.Infof("%s 處理玩家離線 \n", logger.LOG_Main)

	playerIDs, err := mongo.GetDocIDsByFieldValue(mongo.ColName.Player, "onlineState", "Online", mongo.Equal)
	if err != nil {
		fmt.Println("playerOfflineHandler執行mongo.GetDocIDsByFieldValue找Player錯誤:", err)
		return
	}

	if len(playerIDs) <= 0 {
		log.Infof("%s 處理玩家離線完成 \n", logger.LOG_Main)
		return
	}

	// 計算離線閾值時間
	minutesBefore := time.Now().Add(-PLAYER_OFFLINE_THRESHOLD_MINUTES * time.Minute)

	// 批量查詢playerState文檔
	var offlinePlayerStates []mongo.DBPlayerState
	err = mongo.GetDocsByFieldValue(mongo.ColName.PlayerState, "_id", playerIDs, mongo.In, &offlinePlayerStates)
	if err != nil {
		fmt.Println("playerOfflineHandler執行mongo.GetDocIDsByFieldValue找PlayerState錯誤:", err)
		return
	}

	// 找出需要更新的playerIDs
	var offlinePlayerIDs []string
	for _, playerState := range offlinePlayerStates {
		if playerState.LastUpdateAt.Before(minutesBefore) {
			offlinePlayerIDs = append(offlinePlayerIDs, playerState.ID)
		}
	}

	// 批量更新player文件的onlineState
	if len(offlinePlayerIDs) > 0 {
		updateData := bson.D{{Key: "onlineState", Value: "Offline"}}
		_, err := mongo.UpdateDocsByField(mongo.ColName.Player, "_id", offlinePlayerIDs, updateData)
		if err != nil {
			fmt.Println("批量更新 player onlineState 錯誤:", err)
		}
	}

	log.Infof("%s 處理玩家離線完成 \n", logger.LOG_Main)

}