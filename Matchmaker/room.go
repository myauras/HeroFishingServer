package main

import (
	"encoding/json"
	"fmt"
	logger "matchmaker/logger"
	"matchmaker/setting"
	"net"
	"sync"
	"sync/atomic"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	log "github.com/sirupsen/logrus"
	mongo "herofishingGoModule/mongo"
)

type RoomReceptionist struct {
	quickRoomUshers map[string]*Usher // Key值為mapID(不同地圖有不同mapID，用來區分不同房間的玩家不會彼此配對到)
}
type Usher struct {
	roomLock        sync.RWMutex
	rooms           []*room // 已建立的房間
	lastJoinRoomIdx int     // 上一次加房索引，記錄此值避免每次找房間都是從第一間開始找
}
type room struct {
	gameServer *agonesv1.GameServer
	mapID      string        // 地圖ID
	matchType  string        // 配對類型
	maxPlayer  int           //最大玩家數
	players    []*roomPlayer //房間內的玩家
	creater    *roomPlayer   //開房者
	createTime *time.Time    //開房時間
}
type roomPlayer struct {
	id      string        // 玩家ID
	isAuth  bool          // 是否經過帳戶驗證了
	connTCP ConnectionTCP // TCP連線
	mapID   string        // 地圖ID
	room    *room         // 房間資料
}
type ConnectionTCP struct {
	Conn    net.Conn      // TCP連線
	Encoder *json.Encoder // 連線編碼
	Decoder *json.Decoder // 連線解碼
}

func (rr *RoomReceptionist) Init() {
	rr.quickRoomUshers = make(map[string]*Usher)
	//go rr.RoutineCheckOccupiedRoom()
}

// func (rr *RoomReceptionist) RoutineCheckOccupiedRoom() {
// 	timer := time.NewTicker(ROUTINE_CHECK_OCCUPIED_ROOM * time.Minute)
// 	for {
// 		for _, usher := range rr.quickRoomUshers {
// 			usher.CheckOccupiedRoom()
// 		}
// 		<-timer.C
// 	}
// }
// func (u *Usher) CheckOccupiedRoom() {

// }
func (r *room) clearRoom() {

	log.WithFields(log.Fields{
		"players": r.players,
	}).Infof("%s ClearRoom", logger.LOG_ROOM)
	// 清除房間
	for i := 0; i < len(r.players); i++ {
		r.players[i].LeaveRoom()
	}
	r.players = nil
	r.createTime = nil
}

// 玩家離開房間
func (p *roomPlayer) LeaveRoom() {
	p.room = nil
}

func (r *RoomReceptionist) getUsher(mapID string) *Usher {
	usher, ok := r.quickRoomUshers[mapID]
	if !ok {
		newUsher := Usher{}
		r.quickRoomUshers[mapID] = &newUsher
		usher = r.quickRoomUshers[mapID]
	}
	return usher
}

// 加入房間-快速房
func (r *RoomReceptionist) JoinRoom(dbMap mongo.DBMap, player *roomPlayer) *room {

	// 取得房間接待員
	usher := r.getUsher(dbMap.ID)

	// 找等候中的房間
	for i, _ := range usher.rooms {
		roomIdx := (usher.lastJoinRoomIdx + i) % len(usher.rooms)
		room := usher.rooms[roomIdx]
		joined := room.AddPlayer(player)
		// 房間不可加入就換下一間檢查
		if !joined {
			usher.lastJoinRoomIdx = roomIdx
			continue
		}

		log.WithFields(log.Fields{
			"playerID":  player.id,
			"dbMapID":   dbMap.ID,
			"roomIdx":   roomIdx,
			"room":      room,
			"dbMapData": dbMap,
		}).Infof("%s Player join an exist room", logger.LOG_ROOM)

		log.Infof("%s 加入房間= %+v", logger.LOG_Main, room)
		return room
	}

	log.Infof("%s 找不到可加入的房間, 創建一個新房間: %+v", logger.LOG_Main, dbMap)
	// 找不到可加入的房間就創一個新房間
	newCreateTime := time.Now()
	newRoom := room{
		mapID:      dbMap.ID,
		matchType:  dbMap.MatchType,
		maxPlayer:  setting.MAX_PLAYER,
		players:    nil,
		creater:    nil,
		createTime: &newCreateTime,
	}
	// 設定玩家所在地圖
	player.mapID = dbMap.ID
	// 設定玩家為開房者
	newRoom.creater = player
	// 開房者加入此新房
	newRoom.AddPlayer(player)
	// 將新房加到房間清單中
	usher.rooms = append(usher.rooms, &newRoom)
	roomIdx := len(usher.rooms) - 1
	usher.lastJoinRoomIdx = roomIdx

	log.WithFields(log.Fields{
		"playerID":   player.id,
		"dbMapID":    dbMap.ID,
		"roomIdx":    roomIdx,
		"room":       newRoom,
		"dbRoomData": dbMap,
	}).Infof("%s Player create a new room", logger.LOG_ROOM)

	return &newRoom

}

// 檢查此房間是否已經存在該玩家ID
func (r *room) IsIDExist(playerID string) bool {
	for _, v := range r.players {
		if v.id == playerID {
			return true
		}
	}
	return false
}

// 將玩家加入此房間中
func (r *room) AddPlayer(player *roomPlayer) bool {
	// 滿足以下條件之一的房間不可加入
	// 1. 該玩家已在此房間
	// 2. 房間已滿
	if r.IsIDExist(player.id) || len(r.players) >= r.maxPlayer {
		return false
	}

	r.players = append(r.players, player)
	return true
}

// 建立遊戲
func (r *room) CreateGame() error {
	var createGameOK bool
	var err error

	// 產生房間名稱
	roomName, getRoomNameOK := r.generateRoomName()
	if !getRoomNameOK {
		createGameOK = false

		log.WithFields(log.Fields{
			"room": r,
		}).Errorf("%s Generate Room Name Failed", logger.LOG_ROOM)
		err = fmt.Errorf("%s Generate Room Name Failed", logger.LOG_ROOM)
		return err
	}

	log.WithFields(log.Fields{
		"room":     r,
		"roomName": roomName,
	}).Infof("%s Generate Room Name \n", logger.LOG_ROOM)

	// 建立遊戲房
	retryTimes := 0
	timer := time.NewTicker(setting.RETRY_INTERVAL_SECONDS * time.Second)
	for i := 0; i < setting.RETRY_CREATE_GAMESERVER_TIMES; i++ {
		retryTimes = i
		r.gameServer, err = CreateGameServer(roomName, r.getPlayerIDs(), r.creater.id, r.mapID, SelfPodName)
		if err == nil {
			createGameOK = true
			break
		}
		log.Errorf("%s CreateGameServer第%v次失敗: %v", logger.LOG_Main, i, err)
		<-timer.C
	}
	timer.Stop()

	// 寫入建立遊戲房結果Log
	if createGameOK {
		if retryTimes > 0 {
			log.WithFields(log.Fields{
				"retryTimes": retryTimes,
				"error:":     err.Error(),
			}).Infof("%s Create gameServer with retry: \n", logger.LOG_ROOM)
		}
	} else {
		log.WithFields(log.Fields{
			"retryTimes": setting.RETRY_CREATE_GAMESERVER_TIMES,
			"error:":     err.Error(),
		}).Errorf("%s Create gameServer error: \n", logger.LOG_ROOM)
		err = fmt.Errorf("%s Gameserver allocated failed", logger.LOG_ROOM)
	}

	return err
}

var counter int64 // 房間名命名計數器
// 以創房者的id來產生房名
func (r *room) generateRoomName() (string, bool) {
	var roomName string
	if r.creater == nil {
		log.Println("Generating room name failed, creater is nil")
		return roomName, false
	}
	newCounterValue := atomic.AddInt64(&counter, 1)
	roomName = fmt.Sprintf("%s_%d_%s", r.creater.id, newCounterValue, time.Now().Format("20060102T150405"))
	return roomName, true
}
func (r *room) getPlayerIDs() []string {
	ids := []string{}
	for _, v := range r.players {
		ids = append(ids, v.id)
	}
	return ids
}
func (p roomPlayer) playerLeaveRoom() {
	if p.room == nil {
		return
	}
	// 將玩家從房間中移除
	p.room.removePlayer(p)
}

// 將玩家從房間中移除
func (r *room) removePlayer(p roomPlayer) {
	tarIdx := -1
	for i, player := range r.players {
		if player.connTCP == p.connTCP {
			tarIdx = i
		}
	}
	if tarIdx >= 0 {
		newPlayers := []*roomPlayer{}
		for i, v := range r.players {
			if i != tarIdx {
				newPlayers = append(newPlayers, v)
			}
		}
		r.players = newPlayers
	}
	// 如果房間沒人就清除房間
	if len(r.players) <= 0 {
		p.room.clearRoom()
	}
}
