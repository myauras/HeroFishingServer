package main

import (
	"encoding/json"
	"fmt"
	mongo "herofishingGoModule/mongo"
	redis "herofishingGoModule/redis"
	logger "lobby/logger"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

type RequestData struct {
	CMD   string `json:"cmd"`
	Token string `json:"token"`
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

	log.Infof("%s ==============MATCHGAME 啟動==============", logger.LOG_Main)

	router := mux.NewRouter()
	router.HandleFunc("/player/syncredischeck", handleSyncRedisCheck).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// 處理 /player/syncredischeck 路由的 POST 請求
func handleSyncRedisCheck(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	playerID, err := verifyPlayer(requestData.Token)
	if err != nil {
		log.Errorf("%s handleSyncRedisCheck錯誤: %s", logger.LOG_Main, err)
		return
	}

	// 取mongoDB player doc
	var mongoPlayerDoc mongo.DBPlayer
	getPlayerDocErr := mongo.GetDocByID(mongo.ColName.Player, playerID, &mongoPlayerDoc)
	if getPlayerDocErr != nil {
		log.Errorf("%s InitGameRoom時取dbmap資料發生錯誤: %v", logger.LOG_Main, getPlayerDocErr)
		return
	}
	if mongoPlayerDoc.RedisSync { // RedisSync為true就不需要進行資料同步
		return
	}
	// 取redisDB player
	redisPlayer, err := redis.GetPlayerDBData(playerID)
	if err != nil || redisPlayer.ID == "" {
		log.Errorf("%s handleSyncRedisCheck執行redis.GetPlayerDBData錯誤: %s", logger.LOG_Main, err)
		return
	}
	log.Infof("%s 玩家 %s 須同步redisDB資料", logger.LOG_Main, mongoPlayerDoc.ID)

	// 更新玩家mongoDB資料
	updatePlayerBson := bson.D{
		{Key: "point", Value: redisPlayer.Point},     // 設定玩家點數
		{Key: "heroExp", Value: redisPlayer.HeroExp}, // 設定英雄經驗
		{Key: "inMatchgameID", Value: ""},            // 設定玩家不在遊戲房內了
		{Key: "redisSync", Value: true},              // 設定redisSync為true, 代表已經把這次遊玩結果更新上monogoDB了
	}
	mongo.UpdateDocByID(mongo.ColName.Player, mongoPlayerDoc.ID, updatePlayerBson)

	log.Infof("%s 玩家 %s redisDB資料同步完成", logger.LOG_Main, mongoPlayerDoc.ID)
	// // 回傳
	// response := map[string]string{
	// 	"msg":   "success",
	// 	"error": "",
	// }
	// json.NewEncoder(w).Encode(response)
}
func verifyPlayer(token string) (string, error) {
	playerID, err := mongo.PlayerVerify(token)
	if err != nil {
		return "", fmt.Errorf("無效的的Token, 驗證失敗")
	}
	return playerID, nil
}
