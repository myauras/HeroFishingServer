package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"

	"net"
	"sync"
	"time"

	logger "matchmaker/Logger"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	log "github.com/sirupsen/logrus"
)

const (
	RETRY_CREATE_GAMESERVER_TIMES = 2 // 開房失敗時重試X次
	RETRY_INTERVAL_SECONDS        = 3 // 開房失敗重試間隔X秒
	MAX_PLAYER                    = 4 // 房間容納玩家上限為X人
	ROUTINE_CHECK_OCCUPIED_ROOM   = 5 // 每X分鐘檢查佔用房間
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
	roomID     string        // 房間ID
	roomType   string        // 房間類型
	maxPlayer  int           //最大玩家數
	players    []*roomPlayer //房間內的玩家
	createTime *time.Time    //開房時間
}
type roomPlayer struct {
	id      string        // 玩家ID
	isAuth  bool          // 是否經過帳戶驗證了
	conn    net.Conn      // 連線
	encoder *json.Encoder // 連線編碼
	decoder *json.Decoder // 連線解碼
	mapID   string        // 地圖ID
	room    *room         // 房間資料
}
type dbRoomData struct {
	roomID   string `db:"id"`
	roomType string `db:"roomType"`
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
	// 寫LOG
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

// 玩家加入房間
func (r *RoomReceptionist) PlayerJoinQuickRoom(mapID string, roomDataDB dbRoomData, player *roomPlayer) *room {
	usher, ok := r.quickRoomUshers[mapID]
	if !ok {
		newUsher := Usher{}
		r.quickRoomUshers[mapID] = &newUsher
		usher = r.quickRoomUshers[mapID]
	}
	return usher.JoinQuickRoom(mapID, roomDataDB, player)
}

// 配房-快速房
func (u *Usher) JoinQuickRoom(mapID string, roomDataDB dbRoomData, player *roomPlayer) *room {
	// 找等候中的房間

	for i, _ := range u.rooms {
		roomIdx := (u.lastJoinRoomIdx + i) % len(u.rooms)
		room := u.rooms[roomIdx]
		joined := room.joinRoomWithPlayer(player)
		// 房間不可加入就換下一間檢查
		if !joined {
			u.lastJoinRoomIdx = roomIdx
			continue
		}

		// 寫LOG
		log.WithFields(log.Fields{
			"playerID":   player.id,
			"mapID":      mapID,
			"roomIdx":    roomIdx,
			"room":       room,
			"dbRoomData": roomDataDB,
		}).Infof("%s Player join an exist room", logger.LOG_ROOM)
		return room
	}

	// 找不到可加入的房間就創一個新房間
	newCreateTime := time.Now()
	newRoom := room{
		roomID:     roomDataDB.roomID,
		roomType:   roomDataDB.roomType,
		maxPlayer:  MAX_PLAYER,
		players:    nil,
		createTime: &newCreateTime,
	}
	// 開房者加入此新房
	newRoom.joinRoomWithPlayer(player)
	// 將新房加到房間清單中
	u.rooms = append(u.rooms, &newRoom)
	roomIdx := len(u.rooms) - 1
	u.lastJoinRoomIdx = roomIdx

	// 寫LOG
	log.WithFields(log.Fields{
		"playerID":   player.id,
		"waitStr":    mapID,
		"roomIdx":    roomIdx,
		"room":       newRoom,
		"dbRoomData": roomDataDB,
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
func (r *room) joinRoomWithPlayer(player *roomPlayer) bool {
	// 滿足以下條件之一的房間不可加入
	// 1. 該玩家已在此房間
	// 2. 房間已滿
	if r.IsIDExist(player.id) || len(r.players) >= r.maxPlayer {
		return false
	}

	r.players = append(r.players, player)
	return true
}

func (r *room) tryCreateGame() (bool, error) {
	var createGameOK bool
	var err error
	roomName, getRoomNameOK := r.generateRoomName()
	if !getRoomNameOK {
		// 寫LOG
		log.WithFields(log.Fields{
			"room": r,
		}).Errorf("%s Generate Room Name Failed!", logger.LOG_ROOM)
		createGameOK = false
		err = errors.New(fmt.Sprintf("%s Generate Room Name Failed!", logger.LOG_ROOM))
		return createGameOK, err
	}

	// 寫LOG
	log.WithFields(log.Fields{
		"room":     r,
		"roomName": roomName,
	}).Infof("%s Generate Room Name \n", logger.LOG_ROOM)

	// 建立GameServer
	playerUIDs := r.getAllPlayerUID()

	timer := time.NewTicker(RETRY_INTERVAL_SECONDS * time.Second)
	retryTimes := 0
	for i := 0; i < RETRY_CREATE_GAMESERVER_TIMES; i++ {
		retryTimes = i
		r.gameServer, err = CreateGameServer(roomName, playerUIDs, playerUIDs[0], r.roomID, SelfPodName)
		if err == nil {
			createGameOK = true
			break
		}
		<-timer.C
	}
	timer.Stop()

	if createGameOK {
		if retryTimes > 0 {
			// 寫LOG
			log.WithFields(log.Fields{
				"retryTimes": retryTimes,
				"error:":     err.Error(),
			}).Infof("%s Create gameServer with retry: \n", logger.LOG_ROOM)
		}
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"retryTimes": RETRY_CREATE_GAMESERVER_TIMES,
			"error:":     err.Error(),
		}).Errorf("%s Create gameServer error: \n", logger.LOG_ROOM)
		err = errors.New(fmt.Sprintf("%s Gameserver allocated failed", logger.LOG_ROOM))
	}

	return createGameOK, err
}

// 以第一位玩家的id來產生房名
func (r *room) generateRoomName() (string, bool) {
	ok := false
	var roomName string
	for _, v := range r.players {
		md5Data := []byte(v.id + time.Now().String())
		roomName = fmt.Sprintf("%x", md5.Sum(md5Data))
		ok = true
		break
	}

	if ok {
		taipeiLoc, err := time.LoadLocation("Asia/Taipei")
		if err == nil {
			roomName = roomName + time.Now().In(taipeiLoc).Format("20060102")
		}
	}
	return roomName, ok
}
func (r *room) getAllPlayerUID() []string {
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
		if player.conn == p.conn {
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
