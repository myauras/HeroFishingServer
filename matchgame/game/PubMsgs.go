package game

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/redis"
	logger "matchgame/logger"
)

// 送玩家離開訊息給Matchmaker
func (r *Room) PubPlayerLeftMsg(playerID string) {
	publishChannelName := "Game-" + r.RoomName
	playerLeftContent := redis.PlayerLeft{
		PlayerID: playerID,
	}
	contentBytes, err := json.Marshal(playerLeftContent)
	if err != nil {
		log.Errorf("%s PubPlayerLeftMsg序列化PlayerLeft錯誤: %v", logger.LOG_Room, err)
		return
	}
	msg := redis.RedisPubSubPack{
		CMD:     redis.CMD_PLAYERLEFT,
		Content: json.RawMessage(contentBytes),
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("%s PubPlayerLeftMsg序列化RedisPubSubPack錯誤: %s", logger.LOG_Room, err.Error())
		return
	}
	publishErr := redis.Publish(publishChannelName, jsonData)
	if publishErr != nil {
		log.Errorf("%s PubPlayerLeftMsg错误: %s", logger.LOG_Room, publishErr)
		return
	}
	log.Infof("%s 送玩家離開訊息到 %s Msg: %+v", logger.LOG_Room, publishChannelName, msg)
}

// 送房間建立訊息給Matchmaker
func (r *Room) PubGameCreatedMsg() {
	publishChannelName := "Game-" + r.RoomName
	gameCreatedContent := redis.GameCreated{
		MatchgameID: r.DBMatchgame.ID,
	}
	contentBytes, err := json.Marshal(gameCreatedContent)
	if err != nil {
		log.Errorf("%s PubGameCreatedMsg序列化GameCreated錯誤: %v", logger.LOG_Room, err)
		return
	}
	msg := redis.RedisPubSubPack{
		CMD:     redis.CMD_GAMECREATED,
		Content: json.RawMessage(contentBytes),
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("%s PubGameCreatedMsg序列化RedisPubSubPack錯誤: %s", logger.LOG_Room, err.Error())
		return
	}
	publishErr := redis.Publish(publishChannelName, jsonData)
	if publishErr != nil {
		log.Errorf("%s PubGameCreatedMsg错误: %s", logger.LOG_Room, publishErr)
		return
	}
	log.Infof("%s 送遊戲房建立訊息到 %s Msg: %+v", logger.LOG_Room, publishChannelName, msg)
}
