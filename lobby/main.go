package main

import (
	"encoding/json"
	"fmt"
	mongo "herofishingGoModule/mongo"
	redis "herofishingGoModule/redis"
	logger "lobby/logger"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

var Env string // 環境版本

// 請求
type RequestData struct {
	Token     string `json:"token"`
	ValueJson string `json:"valueJson"`
}

func main() {
	// 設定日誌級別
	log.SetLevel(log.InfoLevel)
	// 設定日誌輸出，預設為標準輸出
	log.SetOutput(os.Stdout)
	// 自定義時間格式，包含毫秒
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	log.Infof("%s ==============Lobby 啟動==============", logger.LOG_Main)
	Env = os.Getenv("Env")

	// 初始化MongoDB設定
	mongoAPIPublicKey := os.Getenv("MongoAPIPublicKey")
	mongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey")
	mongoUser := os.Getenv("MongoUser")
	mongoPW := os.Getenv("MongoPW")
	initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey, mongoUser, mongoPW)

	redis.Init() // 初始化Redis

	router := mux.NewRouter()
	router.HandleFunc("/player/syncredischeck", handleSyncRedisCheck).Methods("POST")
	router.HandleFunc("/game/getstate", handleGetState).Methods("POST")

	log.Infof("%s ==============Lobby 啟動完成==============", logger.LOG_Main)
	log.Fatal(http.ListenAndServe(":8080", router))

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

// 處理 /player/syncredischeck 路由的 POST 請求
func handleSyncRedisCheck(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Infof("handleSyncRedisCheck收到msg: %+v", requestData)

	playerID, err := verifyPlayer(requestData.Token)
	if err != nil {
		log.Errorf("%s handleSyncRedisCheck錯誤: %s", logger.LOG_Main, err)
		return
	}

	// 取mongoDB player doc
	var mongoPlayerDoc mongo.DBPlayer
	getPlayerDocErr := mongo.GetDocByID(mongo.ColName.Player, playerID, &mongoPlayerDoc)
	if getPlayerDocErr != nil {
		log.Errorf("%s handleSyncRedisCheck時取dbplayer資料發生錯誤: %v", logger.LOG_Main, getPlayerDocErr)
		return
	}
	log.Infof("%s mongoPlayerDoc.RedisSync: %v", logger.LOG_Main, mongoPlayerDoc.RedisSync)
	if mongoPlayerDoc.RedisSync { // RedisSync為true就不需要進行資料同步
		// 回傳
		response := map[string]string{
			"result": "success",
			"error":  "",
		}
		json.NewEncoder(w).Encode(response)
		log.Infof("%s 玩家 %s 不須同步redisDB資料 redisSync為true", logger.LOG_Main, mongoPlayerDoc.ID)
		return
	}
	// 取redisDB player
	redisPlayer, err := redis.GetPlayerDBData(playerID)
	if err != nil {
		log.Errorf("%s handleSyncRedisCheck執行redis.GetPlayerDBData錯誤: %v", logger.LOG_Main, err)
		return
	}
	if redisPlayer.ID == "" {
		log.Infof("%s 玩家 %s 不須同步redisDB資料 redisPlayer.ID為空", logger.LOG_Main, mongoPlayerDoc.ID)
		// 回傳
		response := map[string]string{
			"result": "success",
			"error":  "",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	log.Infof("%s 玩家 %s 須同步redisDB資料", logger.LOG_Main, mongoPlayerDoc.ID)

	// 更新玩家mongoDB資料
	spellCharges := []int32{redisPlayer.SpellCharge1, redisPlayer.SpellCharge2, redisPlayer.SpellCharge3}
	drops := []int32{redisPlayer.SpellCharge1, redisPlayer.SpellCharge2, redisPlayer.SpellCharge3}
	updatePlayerBson := bson.D{
		{Key: "point", Value: redisPlayer.Point},                       // 玩家點數
		{Key: "pointBuffer", Value: redisPlayer.PointBuffer},           // 玩家點數溢位
		{Key: "totalWin", Value: redisPlayer.TotalWin},                 // 玩家總贏點數
		{Key: "totalExpenditure", Value: redisPlayer.TotalExpenditure}, // 玩家總花費點數
		{Key: "leftGameAt", Value: time.Now()},                         // 離開遊戲時間
		{Key: "inMatchgameID", Value: ""},                              // 玩家不在遊戲房內了
		{Key: "heroExp", Value: redisPlayer.HeroExp},                   // 英雄經驗
		{Key: "spellCharges", Value: spellCharges},                     // 技能充能
		{Key: "drops", Value: drops},                                   // 掉落道具
		{Key: "redisSync", Value: true},                                // 設定redisSync為true, 代表已經把這次遊玩結果更新上monogoDB了
	}
	_, updateErr := mongo.UpdateDocByBsonD(mongo.ColName.Player, mongoPlayerDoc.ID, updatePlayerBson)
	if updateErr != nil {
		log.Errorf("%s 更新玩家 %s DB資料錯誤: %v", logger.LOG_Main, mongoPlayerDoc.ID, updateErr)
	} else {
		log.Infof("%s 玩家 %s redisDB資料同步完成", logger.LOG_Main, mongoPlayerDoc.ID)
	}

	// 回傳
	response := map[string]string{
		"result": "success",
		"error":  "",
	}
	json.NewEncoder(w).Encode(response)
}

// 處理 /game/getstate 路由的 POST 請求
func handleGetState(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var mongoGameStateDoc mongo.DBGameState
	getGameStateDocErr := mongo.GetDocByID(mongo.ColName.GameSetting, "GameState", &mongoGameStateDoc)
	if getGameStateDocErr != nil {
		log.Errorf("%s 取GameState資料發生錯誤: %v", logger.LOG_Main, getGameStateDocErr)
		return
	}
	log.Infof("%v", mongoGameStateDoc.IosReview)
	iosReview := strconv.FormatBool(mongoGameStateDoc.IosReview)
	// 回傳
	response := map[string]string{
		"result": "success",
		"error":  "",
		"data":   iosReview,
	}
	json.NewEncoder(w).Encode(response)
}

func verifyPlayer(token string) (string, error) {
	playerID, err := mongo.PlayerVerify(token)
	if err != nil {
		return "", fmt.Errorf("無效的的Token: %s", token)
	}
	return playerID, nil
}
