package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	MAX_PLAYER                  = 4 // 房間容納玩家上限為X人
	ROUTINE_CHECK_OCCUPIED_ROOM = 5 // 每X分鐘檢查佔用房間
)

type RoomReceptionist struct {
	quickRoomUshers map[string]*Usher // Key值為mapID(不同地圖有不同mapID，用來區分不同房間的玩家不會彼此配對到)
}
type Usher struct {
	roomLock        sync.RWMutex
	Rooms           []*room // 已建立的房間
	lastJoinRoomIdx int     // 上一次加房索引，記錄此值避免每次找房間都是從第一間開始找
}
type room struct {
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
	go rr.RoutineCheckOccupiedRoom()
}
func (rr *RoomReceptionist) RoutineCheckOccupiedRoom() {
	timer := time.NewTicker(ROUTINE_CHECK_OCCUPIED_ROOM * time.Minute)
	for {
		for _, usher := range rr.quickRoomUshers {
			usher.CheckOccupiedRoom()
		}
		<-timer.C
	}
}
func (u *Usher) CheckOccupiedRoom() {
	// for _, room := range u.Rooms {
	// 	if room.isStart {
	// 		// 寫LOG
	// 		log.WithFields(log.Fields{
	// 			"room": room,
	// 		}).Infof("%s ClearOccupiedRoom", LOG_ROOM)

	// 		// 清除房間
	// 		room.clearRoom()
	// 	}
	// }
}
func (r *room) clearRoom() {
	// 寫LOG
	log.WithFields(log.Fields{
		"players": r.players,
	}).Infof("%s ClearRoom", LOG_ROOM)
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

	for i, _ := range u.Rooms {
		roomIdx := (u.lastJoinRoomIdx + i) % len(u.Rooms)
		room := u.Rooms[roomIdx]
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
		}).Infof("%s Player join an exist room", LOG_ROOM)
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
	u.Rooms = append(u.Rooms, &newRoom)
	roomIdx := len(u.Rooms) - 1
	u.lastJoinRoomIdx = roomIdx

	// 寫LOG
	log.WithFields(log.Fields{
		"playerID":   player.id,
		"waitStr":    mapID,
		"roomIdx":    roomIdx,
		"room":       newRoom,
		"dbRoomData": roomDataDB,
	}).Infof("%s Player create a new room", LOG_ROOM)

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

func (r *room) TryStartGame() (bool, error) {
	var startOK bool
	var err error
	roomName, ok := r.generateRoomName()
	if !ok {

		// 寫LOG
		log.WithFields(log.Fields{
			"room": r,
		}).Errorf("%s Generate Room Name Failed!", LOG_ROOM)
		startOK = false
		err = errors.New(fmt.Sprintf("%d Generate Room Name Failed!", LOG_ROOM))
		return startOK, err
	}

	// 寫LOG
	log.WithFields(log.Fields{
		"room":     r,
		"roomName": roomName,
	}).Infof("%s Generate Room Name \n", LOG_ROOM)

	playerUIDs := r.getAllPlayerUID(true)
	r.gameServer, err = CreateGameServer(roomName, playerUIDs, playerUIDs[0], r.gamedataRoomUID, SelfPodName)
	if err != nil {
		retryNum := 0
		retryOK := false
		timer := time.NewTicker(RETRY_GAMESERVER_DURATION * time.Second)
		for retryNum < RETRY_GAMESERVER_TIMES {
			r.gameServer, err = CreateGameServer(roomName, playerUIDs, playerUIDs[0], r.gamedataRoomUID, SelfPodName)
			if err == nil {
				retryOK = true
				break
			}
			<-timer.C
			retryNum++
		}
		log.Printf("CreateGameServer error: %s, RetryTime: %d, Result: %t.\n", err.Error(), retryNum+1, retryOK)
		if !retryOK {
			startOK = false
			err = errors.New("GAMESERVER_ALLOCATED_FAILED")
			return startOK, err
		}
	}
	startOK = true

	return startOK, err
}

func (r *room) generateRoomName() (string, bool) {
	gotID := false
	var roomName string
	for _, v := range r.players {
		md5Data := []byte(v.id + time.Now().String())
		roomName = fmt.Sprintf("%x", md5.Sum(md5Data))
		gotID = true
		break
	}

	if gotID {
		taipeiLoc, err := time.LoadLocation("Asia/Taipei")
		if err == nil {
			roomName = roomName + time.Now().In(taipeiLoc).Format("20060102")
		}
	}
	return roomName, gotID
}
func (r *room) getAllPlayerUID(showAI bool) []string {
	ids := []string{}
	for _, v := range r.players {
		ids = append(ids, v.id)
	}
	return ids
}
