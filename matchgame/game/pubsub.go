package game

import (
	"encoding/json"
	"herofishingGoModule/redis"
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 送玩家離開訊息給Matchmaker
func (r *Room) PubPlayerLeftMsg(playerID string) {
	if Mode == "non-agones" {
		return
	}
	publishChannelName := "Matchgame-" + r.RoomName
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
func (r *Room) PubGameCreatedMsg(packID int) {
	if Mode == "non-agones" {
		return
	}
	publishChannelName := "Matchgame-" + r.RoomName
	gameCreatedContent := redis.GameCreated{
		MatchgameID: r.DBMatchgame.ID,
		PackID:      packID,
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
	log.Errorf("%s 送遊戲房建立訊息到 %s Msg: %+v", logger.LOG_Room, publishChannelName, msg)
}

// 訂閱Redis Matchmaker訊息
func (r *Room) SubMatchmakerMsg() {
	channelName := "Matchmaker-" + r.RoomName
	log.Infof("%s 訂閱Redis Matchmaker(%s)", logger.LOG_Room, channelName)
	msgChan := make(chan interface{})
	err := redis.Subscribe(channelName, msgChan)
	if err != nil {
		log.Errorf("%s 訂閱錯誤: %s", logger.LOG_Room, err)
		return
	}

	for msg := range msgChan {
		var data redis.RedisPubSubPack
		byteMsg := []byte(msg.(string))
		err := json.Unmarshal(byteMsg, &data)
		if err != nil {
			log.Errorf("%s JSON解析錯誤: %s", logger.LOG_Room, err)
			continue
		}

		switch data.CMD {
		case redis.CMD_KICK_DISCONNECTED_PLAYER: // 玩家A跑去加新遊戲房, 將殘留的已斷線玩家A踢掉
			var kickDisconnectedPlayerData redis.KickDisconnectedPlayer
			err := json.Unmarshal(data.Content, &kickDisconnectedPlayerData)
			if err != nil {
				log.Errorf("%s SubRoomMsg JSON 解析 Content(%s) 錯誤: %v", logger.LOG_Room, data.CMD, err)
				continue
			}
			player := r.GetPlayerByID(kickDisconnectedPlayerData.PlayerID)
			if player != nil {
				r.KickPlayer(player.ConnTCP.Conn, "將殘留的已斷線玩家踢掉")
			}
			log.Infof("%s 收到Matchmaker將殘留的已斷線玩家踢掉: %s", logger.LOG_Room, kickDisconnectedPlayerData.PlayerID)
		}
	}
}
