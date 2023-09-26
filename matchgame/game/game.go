package game

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const CHAN_BUFFER = 4

var EnvVersion string          // 環境版本
var room Room                  // 房間
var UPDATE_INTERVAL_MS float64 // 每X毫秒更新一次

type DBMap struct {
	ID     string // DB文件ID
	Bet    string // 地圖下注倍率
	Enable bool   // 是否開放
}

func InitGameRoom(firebaseDocID string, roomName string, playerIDs [PLAYER_NUMBER]string, outputPlayerUIDs [PLAYER_NUMBER]string, gameSetting DBMap, waitRoom chan *Room, serverName string) {
	if room.RoomName != "" {
		return
	}

	if UPDATE_INTERVAL_MS <= 0 {
		log.Println("Error Setting UDP Update interval.")
		UPDATE_INTERVAL_MS = 200
	}

	room.Init(gameSetting)
	room.SetDocumentID(firebaseDocID)
	room.SetRoomName(roomName)
	room.SetPlayers(playerIDs, outputPlayerUIDs)
	var logUIDs [PLAYER_NUMBER]string
	logUIDs = playerIDs
	if AI_LOAD_STORAGE {
		logUIDs = outputPlayerUIDs
	}

	createRoomLogData := map[string]interface{}{
		"UID":            roomName,
		"PlayerList":     logUIDs,
		"SettlementRoom": gameSetting.GameDataRoomUID,
		"CreateTime":     time.Now(),
		"Bet":            gameSetting.Bet,
		"ThinkTime":      gameSetting.ThinkTime,
		"ServerName":     serverName,
	}
	//FirebaseFunction.LogCreateGameRoom(createRoomLogData)
	FirebaseFunction.LogCreateGameRoomByRoomName(roomName, createRoomLogData)
	fmt.Println("InitGameRoom RoomType: ", gameSetting.RoomType)
	if gameSetting.RoomType == "Friend" {
		FirebaseFunction.AddCreateFriendRoomTimes(playerIDs[0])
	}
	waitRoom <- &room
}
func (r *Room) WriteGameErrorLog(log string) {
	r.ErrorLogs = append(r.ErrorLogs, log)
}
