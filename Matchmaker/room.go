package main

import (
	"encoding/json"
	"fmt"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/setting"
	"herofishingGoModule/utility"
	logger "matchmaker/logger"
	mSetting "matchmaker/setting"
	"net"
	"sync"
	"sync/atomic"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	log "github.com/sirupsen/logrus"
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
	gameServer    *agonesv1.GameServer
	dbMapID       string        // DB地圖ID
	dbMatchgameID string        // 就是RoomName由Matchmaker產生，格視為[玩家ID]_[累加數字]_[日期時間]
	matchType     string        // 配對類型
	players       []*roomPlayer // 房間內的玩家
	creater       *roomPlayer   // 開房者
	createTime    *time.Time    // 開房時間
}
type roomPlayer struct {
	id      string        // 玩家ID
	isAuth  bool          // 是否經過帳戶驗證了
	connTCP ConnectionTCP // TCP連線
	dbMapID string        // 地圖ID
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
	}).Infof("%s ClearRoom", logger.LOG_Room)
	// 清除房間
	for i, v := range r.players {
		if v == nil {
			continue
		}
		r.players[i].LeaveRoom()
	}
	r.players = nil
	r.createTime = nil
}

// 玩家離開房間
func (p *roomPlayer) LeaveRoom() {
	p.room = nil
}

// 取得房間分配者
func (r *RoomReceptionist) getUsher(dbMapID string) *Usher {
	usher, ok := r.quickRoomUshers[dbMapID]
	if !ok {
		newUsher := Usher{}
		r.quickRoomUshers[dbMapID] = &newUsher
		usher = r.quickRoomUshers[dbMapID]
	}
	return usher
}

// 玩家要加入/建立房間之前, 檢查所有房間如果該玩家斷線但還殘留在房間中, 通知Matchgame踢出該玩家
func (r *RoomReceptionist) KickDisconnectedPlayer(palyerID string) {
	for _, v := range r.quickRoomUshers {
		for _, room := range v.rooms {
			if room.IsIDExist(palyerID) {
				room.PubKickDisconnectedPlayer(palyerID)
				room.RemovePlayer(palyerID) // 將該玩家從房間中移除
				return
			}
		}
	}
}

// 加入房間-快速房, 回傳房間與是否為新開房間
func (r *RoomReceptionist) JoinRoom(packID int, dbMap mongo.DBMap, player *roomPlayer) (*room, bool) {
	r.KickDisconnectedPlayer(player.id) // 玩家要加入/建立房間之前, 檢查所有房間如果該玩家斷線但還殘留在房間中, 通知Matchgame踢出該玩家
	// 取得房間接待員
	usher := r.getUsher(dbMap.ID)
	// 找空房間
	for i, _ := range usher.rooms {
		roomIdx := (usher.lastJoinRoomIdx + i) % len(usher.rooms)
		room := usher.rooms[roomIdx]
		joined := room.AddPlayer(player)
		// 房間不可加入就換下一間檢查
		if !joined {
			usher.lastJoinRoomIdx = roomIdx
			continue
		}

		// 確認目Matchgame Server是否還活著, 掛掉就繼續找下一個
		err := CheckGameServer(room.dbMatchgameID)
		if err != nil {
			utility.RemoveFromSliceByIdx(usher.rooms, roomIdx) // 移除掛掉的房間
			log.Errorf("%s 目標遊戲房已掛: %v", logger.LOG_Room, err)
			continue
		}

		log.WithFields(log.Fields{
			"playerID":  player.id,
			"dbMapID":   dbMap.ID,
			"roomIdx":   roomIdx,
			"room":      room,
			"dbMapData": dbMap,
		}).Infof("%s Player join an exist room", logger.LOG_Room)

		log.Errorf("%s 玩家 %s 加入房間(%v/%v) 房間資料: %+v", logger.LOG_Room, player.id, room.PlayerCount(), setting.PLAYER_NUMBER, room)
		return room, false
	}

	log.Errorf("%s 玩家 %s 找不到可加入的房間, 創建一個新房間(%v/%v): %+v", logger.LOG_Room, player.id, 1, setting.PLAYER_NUMBER, dbMap)
	// 找不到可加入的房間就創一個新房間
	newCreateTime := time.Now()
	newRoom := room{
		dbMapID:    dbMap.ID,
		matchType:  dbMap.MatchType,
		players:    nil,
		creater:    nil,
		createTime: &newCreateTime,
	}
	// 設定玩家所在地圖
	player.dbMapID = dbMap.ID
	// 設定玩家為開房者
	newRoom.creater = player
	// 開房者加入此新房
	newRoom.AddPlayer(player)
	// 將新房加到房間清單中
	usher.rooms = append(usher.rooms, &newRoom)
	roomIdx := len(usher.rooms) - 1
	usher.lastJoinRoomIdx = roomIdx

	// 建立遊戲(Matchgame Server)
	err := player.room.CreateGame(packID)
	if err != nil {
		log.Errorf("%s 建立Matchgame server失敗: %v", logger.LOG_Room, err)
		return nil, false
	}

	log.WithFields(log.Fields{
		"playerID":   player.id,
		"dbMapID":    dbMap.ID,
		"roomIdx":    roomIdx,
		"room":       newRoom,
		"dbRoomData": dbMap,
	}).Infof("%s Player create a new room", logger.LOG_Room)

	return &newRoom, true

}

// 檢查此房間是否已經存在該玩家ID
func (r *room) IsIDExist(playerID string) bool {
	for _, v := range r.players {
		if v == nil {
			continue
		}
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
	if r.IsIDExist(player.id) || len(r.players) >= setting.PLAYER_NUMBER {
		return false
	}
	player.room = r                       // 將玩家的房間設定為此房間
	r.players = append(r.players, player) // 將房間的玩家清單加入此玩家
	return true
}

// 將玩家從房間中移除
func (r *room) RemovePlayer(playerID string) {
	for i, v := range r.players {
		if v != nil && v.id == playerID {
			r.players[i] = nil
			break
		}
	}
}

// 建立遊戲
func (r *room) CreateGame(packID int) error {
	var err error
	if r == nil {
		err = fmt.Errorf("%s CreateGame Room的r為nil", logger.LOG_Room)
		return err
	}
	var createGameOK bool

	// 產生房間名稱
	roomName, getRoomNameOK := r.generateRoomName()
	if !getRoomNameOK {
		createGameOK = false

		log.WithFields(log.Fields{
			"room": r,
		}).Errorf("%s Generate Room Name Failed", logger.LOG_Room)
		err = fmt.Errorf("%s Generate Room Name Failed", logger.LOG_Room)
		return err
	}

	r.dbMatchgameID = roomName
	log.WithFields(log.Fields{
		"room":     r,
		"roomName": roomName,
	}).Infof("%s Generate Room Name \n", logger.LOG_Room)

	// 建立遊戲房
	retryTimes := 0
	timer := time.NewTicker(mSetting.RETRY_INTERVAL_SECONDS * time.Second)
	for i := 0; i < mSetting.RETRY_CREATE_GAMESERVER_TIMES; i++ {
		retryTimes = i
		r.gameServer, err = CreateGameServer(packID, roomName, r.GetPlayerIDs(), r.creater.id, r.dbMapID, SelfPodName)
		if err == nil {
			createGameOK = true
			break
		}
		log.Errorf("%s CreateGameServer第%v次失敗: %v", logger.LOG_Room, i, err)
		<-timer.C
	}
	timer.Stop()

	// 寫入建立遊戲房結果Log
	if createGameOK {
		if retryTimes > 0 {
			log.WithFields(log.Fields{
				"retryTimes": retryTimes,
				"error:":     err.Error(),
			}).Infof("%s Create gameServer with retry: \n", logger.LOG_Room)
		}
		go r.SubMatchgameMsg() // 訂閱房間資訊
	} else {
		log.WithFields(log.Fields{
			"retryTimes": mSetting.RETRY_CREATE_GAMESERVER_TIMES,
			"error:":     err.Error(),
		}).Errorf("%s Create gameServer error: \n", logger.LOG_Room)
		err = fmt.Errorf("%s Gameserver allocated failed", logger.LOG_Room)
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

// 取遊戲房中玩家ID清單 如果該位置沒有玩家會存空字串
func (r *room) GetPlayerIDs() []string {
	ids := []string{}
	for _, v := range r.players {
		if v == nil {
			ids = append(ids, "")
			continue
		}
		ids = append(ids, v.id)
	}
	return ids
}

// 取得房間玩家數
func (r *room) PlayerCount() int {
	count := 0
	for _, v := range r.players {
		if v == nil {
			continue
		}
		count++
	}
	return count
}
