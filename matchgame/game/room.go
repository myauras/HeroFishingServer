package game

import (
	"errors"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	"matchgame/packet"
	"net"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type GameState int // 目前遊戲狀態列舉

const (
	Init GameState = iota
	Start
	End
)

const MAX_ALLOW_DISCONNECT_SECS float64 = 20.0 // 最長允許玩家斷線X秒

type Room struct {
	// 玩家陣列(索引0~3 分別代表4個玩家)
	// 1. 索引代表玩家座位
	// 2. 座位無關玩家進來順序 有人離開就會空著 例如 索引2的玩家離開 players[2]就會是nil 直到有新玩家加入
	players                [setting.PLAYER_NUMBER]Player // 玩家陣列
	RoomName               string                        // 房間名稱(也是DB文件ID)(房主UID+時間轉 MD5)
	gameState              GameState                     // 遊戲狀態
	DBMatchgame            *mongo.DBMatchgame            // DB遊戲房資料
	DBmap                  *mongo.DBMap                  // DB地圖設定
	GameTime               float64                       // 遊戲開始X秒
	MaxAllowDisconnectSecs float64                       // 最長允許玩家斷線秒數
	ErrorLogs              []string                      // ErrorLogs
	lastChangeStateTime    time.Time                     // 上次更新房間狀態時間
	MutexLock              sync.Mutex
}

const CHAN_BUFFER = 4

var Env string                       // 環境版本
var MyRoom Room                      // 房間
var UPDATE_INTERVAL_MS float64 = 100 // 每X毫秒更新一次

func InitGameRoom(dbMapID string, playerIDs [setting.PLAYER_NUMBER]string, roomName string, ip string, port int32, podName string, nodeName string, matchmakerPodName string, roomChan chan *Room) {
	if MyRoom.RoomName != "" {
		return
	}

	if UPDATE_INTERVAL_MS <= 0 {
		log.Errorf("%s Error Setting UDP Update interval", logger.LOG_Room)
		UPDATE_INTERVAL_MS = 100
	}

	// 依據dbMapID從DB中取dbMap設定
	log.Infof("%s 取DBMap資料", logger.LOG_Room)
	var dbMap mongo.DBMap
	err := mongo.GetDocByID(mongo.ColName.Map, dbMapID, &dbMap)
	if err != nil {
		log.Errorf("%s InitGameRoom時取dbmap資料發生錯誤", logger.LOG_Room)
	}
	log.Infof("%s 取DBMap資料成功 DBMapID: %s JsonMapID: %v", logger.LOG_Room, dbMap.ID, dbMap.JsonMapID)

	// 設定dbMatchgame資料
	var dbMatchgame mongo.DBMatchgame
	dbMatchgame.ID = roomName
	dbMatchgame.CreatedAt = time.Now()
	dbMatchgame.DBMapID = dbMapID
	dbMatchgame.PlayerIDs = playerIDs
	dbMatchgame.IP = ip
	dbMatchgame.Port = port
	dbMatchgame.NodeName = nodeName
	dbMatchgame.PodName = podName
	dbMatchgame.MatchmakerPodName = matchmakerPodName

	// 初始化房間設定
	MyRoom.RoomName = roomName
	MyRoom.gameState = Init
	MyRoom.DBmap = &dbMap
	MyRoom.DBMatchgame = &dbMatchgame
	MyRoom.GameTime = 0
	MyRoom.MaxAllowDisconnectSecs = MAX_ALLOW_DISCONNECT_SECS

	// 這裡之後要加房間初始化Log到DB

	log.Infof("%s Init room", logger.LOG_Room)
	roomChan <- &MyRoom
}
func (r *Room) WriteGameErrorLog(log string) {
	r.ErrorLogs = append(r.ErrorLogs, log)
}

// 玩家加入房間 成功時回傳true
func (r *Room) PlayerJoin(player Player) bool {
	index := -1
	for i, v := range r.players {
		if v.ID == player.ID { // 如果要加入的玩家ID與目前房間的玩家ID一樣就回傳失敗
			log.Errorf("%s PlayerJoin failed, room exist the same playerID: %v.\n", logger.LOG_Room, player.ID)
			return false
		}
		if index == -1 && v.ID == "" { // 有座位是空的就把座位索引存起來
			index = i
			break
		}
	}

	if index == -1 { // 沒有找到座位代表房間滿人
		log.Errorf("%s PlayerJoin failed, room is full", logger.LOG_Room)
		return false
	}

	// 設定玩家
	r.players[index] = player
	return true
}

// 玩家離開房間
func (r *Room) PlayerLeave(conn net.Conn) {
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()
	seatIndex := r.getPlayerIndex(conn) // 取得座位索引
	if seatIndex < 0 {
		log.Errorf("%s PlayerLeave failed, get player seat failed", logger.LOG_Room)
		return
	}
	r.players[seatIndex].CloseConnection()
	r.UpdatePlayerStatus()
}

func (r *Room) HandleMessage(conn net.Conn, packet packet.Pack, stop chan struct{}) error {
	seatIndex := r.getPlayerIndex(conn)
	if seatIndex == -1 {
		log.Errorf("%s HandleMessage fialed, Player is not in connection list", logger.LOG_Room)
		return errors.New("HandleMessage fialed, Player is not in connection list")
	}
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()
	conn.SetDeadline(time.Time{}) // 移除連線超時設定
	// 處理各類型封包
	switch packet.CMD {
	case "CMD類型":
	}
	return nil
}

// 取得玩家座位索引
func (r *Room) getPlayerIndex(conn net.Conn) int {
	for i, v := range r.players {
		if v.ConnTCP.Conn == conn {
			return i
		}
	}
	return -1
}

// 開始遊戲房主循環
func (r *Room) StartRun(stop chan struct{}, endGame chan struct{}) {
	go r.gameStateLooop(stop, endGame)
	go r.UpdateTimer(stop)
	go r.StuckCheck(stop)
}

func (r *Room) StuckCheck(stop chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s StuckCheck error: %v.\n%s", logger.LOG_Room, err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	timer := time.NewTicker(15 * time.Second)
	r.lastChangeStateTime = time.Now()
	for {
		<-timer.C
		elapsed := time.Since(r.lastChangeStateTime)
		if (elapsed.Minutes()) >= 3 && r.gameState != End {
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			r.WriteGameErrorLog("StuckCheck")
			stop <- struct{}{}
		}
	}
}

// 遊戲狀態循環
func (r *Room) gameStateLooop(stop chan struct{}, endGame chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s gameUpdate error: %v.\n%s", logger.LOG_Room, err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	for {
		switch r.gameState {
		case Init:
		case End:
		}
		r.lastChangeStateTime = time.Now()
		select {
		case <-stop:
			r.ChangeState(End)
		default:
		}
	}
}

// 改變遊戲狀態
func (r *Room) ChangeState(state GameState) {
	r.gameState = state
}

// 送封包給遊戲房間內所有玩家
func (r *Room) broadCastPacket(pack *packet.Pack) {
	anyError := false

	// 送封包給所有房間中的玩家
	for _, v := range r.players {
		if v.ConnTCP.Conn == nil {
			continue
		}
		err := packet.SendPack(v.ConnTCP.Encoder, pack)
		if err != nil {
			log.Errorf("%s broadCastPacket錯誤: %v", logger.LOG_Room, err)
			anyError = true
		}
	}
	// 有錯誤就重送封包
	if anyError {
		r.UpdatePlayerStatus()
	}
}

// 送封包給玩家
func (r *Room) sendPacketToPlayer(pIndex int, pack *packet.Pack) {
	if r.players[pIndex].ConnTCP.Conn == nil {
		return
	}
	err := packet.SendPack(r.players[pIndex].ConnTCP.Encoder, pack)
	if err != nil {
		log.Errorf("%s SendPacketToPlayer error: %v", logger.LOG_Room, err)
		r.players[pIndex].CloseConnection()
		r.UpdatePlayerStatus()
	}
}

// 取得遊戲房中所有玩家狀態
func (r *Room) GetPlayerStatus() [setting.PLAYER_NUMBER]PlayerStatus {
	playerStatuss := [setting.PLAYER_NUMBER]PlayerStatus{}
	for i, v := range r.players {
		playerStatuss[i] = *v.Status
	}
	return playerStatuss
}

// 送遊戲房中所有玩家狀態封包
func (r *Room) UpdatePlayerStatus() {
	r.broadCastPacket(&packet.Pack{
		CMD:    packet.UPDATE_GAME_STATE_REPLY,
		PackID: -1,
		Content: &UpdateRoomContent{
			PlayerStatuss: r.GetPlayerStatus(),
		},
	})
}

// 更新玩家離開時間
func (r *Room) UpdateTimer(stop chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s UpdatePlayerLeaveTime error: %v.\n%s", logger.LOG_Room, err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	ticker := time.NewTicker(time.Duration(200) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			r.MutexLock.Lock()
			r.GameTime += UPDATE_INTERVAL_MS / 1000
			for _, player := range r.players {
				if player.ConnTCP.Conn == nil {
					player.LeftSecs += UPDATE_INTERVAL_MS / 1000
					// if r.players[i].LeftSecs < MAX_ALLOW_DISCONNECT_SECS {
					// 	r.players[i].LeftSecs += UPDATE_INTERVAL_MS / 1000
					// } else {
					// 	r.players[i].LeftSecs = MAX_ALLOW_DISCONNECT_SECS
					// }
				} else {
					player.LeftSecs = 0
				}
			}
			r.MutexLock.Unlock()
		case <-stop:
			return
		}
	}
}
