package main

import (
	"encoding/json"
	"herofishingGoModule/redis"

	logger "matchmaker/logger"
	"matchmaker/packet"

	log "github.com/sirupsen/logrus"
)

// 訂閱Redis Matchgame房間訊息
func (r *room) SubMatchgameMsg() {
	channelName := "Matchgame-" + r.dbMatchgameID
	log.Infof("%s 訂閱Redis Matchgame(%s)", logger.LOG_Room, channelName)
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
		case redis.CMD_PLAYERLEFT: // 玩家離開
			var playerLeftData redis.PlayerLeft
			err := json.Unmarshal(data.Content, &playerLeftData)
			if err != nil {
				log.Errorf("%s SubRoomMsg JSON 解析 Content(%s) 錯誤: %v", logger.LOG_Room, data.CMD, err)
				continue
			}
			r.RemovePlayer(playerLeftData.PlayerID) // 將該玩家從房間中移除
			log.Infof("%s 收到Matchgame玩家離開: %s", logger.LOG_Room, playerLeftData.PlayerID)
		case redis.CMD_GAMECREATED: // 房間建立
			var gameCreated redis.GameCreated
			err := json.Unmarshal(data.Content, &gameCreated)
			if err != nil {
				log.Errorf("%s SubRoomMsg JSON 解析 Content(%s) 錯誤: %v", logger.LOG_Room, data.CMD, err)
				continue
			}
			log.Errorf("%s 收到Matchgame房間建立完成: %s", logger.LOG_Room, gameCreated.MatchgameID)
			creater := r.players[0]
			if creater == nil {
				return
			}
			packErr := packet.SendPack(creater.connTCP.Encoder, &packet.Pack{
				CMD:    packet.CREATEROOM_TOCLIENT,
				PackID: gameCreated.PackID,
				Content: &packet.CreateRoom_ToClient{
					CreaterID:     creater.id,
					PlayerIDs:     r.GetPlayerIDs(),
					DBMapID:       r.dbMapID,
					DBMatchgameID: r.dbMatchgameID,
					IP:            r.gameServer.Status.Address,
					Port:          r.gameServer.Status.Ports[0].Port,
					PodName:       r.gameServer.ObjectMeta.Name,
				},
			})
			if packErr != nil {
				return
			}

		}
	}
}

// 通知Matchgame踢出殘留的已斷線玩家
func (r *room) PubKickDisconnectedPlayer(playerID string) {
	publishChannelName := "Matchmaker-" + r.dbMatchgameID
	gameCreatedContent := redis.KickDisconnectedPlayer{
		PlayerID: playerID,
	}
	contentBytes, err := json.Marshal(gameCreatedContent)
	if err != nil {
		log.Errorf("%s PubGameCreatedMsg序列化%s錯誤: %v", logger.LOG_Room, redis.CMD_KICK_DISCONNECTED_PLAYER, err)
		return
	}
	msg := redis.RedisPubSubPack{
		CMD:     redis.CMD_KICK_DISCONNECTED_PLAYER,
		Content: json.RawMessage(contentBytes),
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("%s PubGameCreatedMsg序列化RedisPubSubPack錯誤: %s", logger.LOG_Room, err.Error())
		return
	}
	publishErr := redis.Publish(publishChannelName, jsonData)
	if publishErr != nil {
		log.Errorf("%s %s错误: %s", logger.LOG_Room, redis.CMD_KICK_DISCONNECTED_PLAYER, publishErr)
		return
	}
	log.Errorf("%s 通知Matchgame(%s)踢出殘留的已斷線玩家 Msg: %+v", logger.LOG_Room, publishChannelName, msg)
}
