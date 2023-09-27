package game

import (
	log "github.com/sirupsen/logrus"
	logger "matchgame/logger"
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

func InitGameRoom(serverName string, dbMapID string, roomName string, player Player, waitRoom chan *Room) {
	if room.RoomName != "" {
		return
	}

	if UPDATE_INTERVAL_MS <= 0 {
		log.Errorf("%s Error Setting UDP Update interval", logger.LOG_Game)
		UPDATE_INTERVAL_MS = 200
	}

	// 依據dbMapID從DB中取dbMap設定
	dbMap := DBMap{}
	room.Init(roomName, dbMap, player)

	// 這裡之後要加房間初始化Log到DB

	log.Infof("%s Init room", logger.LOG_Game)
	waitRoom <- &room
}
func (r *Room) WriteGameErrorLog(log string) {
	r.ErrorLogs = append(r.ErrorLogs, log)
}
