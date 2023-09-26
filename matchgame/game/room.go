package game

import (
	"encoding/json"
	"errors"
	"fmt"
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

const PLAYER_NUMBER int = 4 //房間最大玩家數量

type GameState int // 目前遊戲狀態列舉

const (
	Init GameState = iota
	Start
	End
)

const MAX_ALLOW_DISCONNECT_SECS float64 = 20.0 // 最長允許玩家斷線秒數

type Room struct {
	RoomName               string                // 房間名稱(也是DB文件ID)(房主UID+時間轉 MD5)
	gameState              GameState             // 遊戲狀態
	CreaterID              string                // 開房者ID
	players                [PLAYER_NUMBER]Player // 玩家陣列(也是座位0~3 分別代表4個玩家)
	DBmap                  DBMap                 // DB地圖設定
	ServerIP               string                // ServerIP
	ServerPort             int                   // ServerPort
	PassSecs               float64               // 遊戲開始X秒
	MaxAllowDisconnectSecs float64               // 最長允許玩家斷線秒數
	ErrorLogs              []string              // ErrorLogs
	lastChangeStateTime    time.Time             // 上次更新房間狀態時間
	MutexLock              sync.Mutex
}

// 初始化房間
func (r *Room) Init(roomName string, dbMap DBMap) {

	// 初始化房間設定
	r.MaxAllowDisconnectSecs = MAX_ALLOW_DISCONNECT_SECS
	r.PassSecs = 0

	// 初始化DB中的地圖設定
	r.RoomName = roomName
	r.DBmap = dbMap
}

// 玩家加入房間 成功時回傳true
func (g *Room) PlayerJoin(conn net.Conn, encoder *json.Encoder, decoder *json.Decoder, playerID string) bool {
	index := -1
	for i, v := range g.players {
		if v.ID == playerID { // 如果要加入的玩家ID與目前房間的玩家ID一樣就回傳失敗
			log.Errorf("%s PlayerJoin failed, room exist the same playerID: %v.\n", logger.LOG_Room, playerID)
			return false
		}
		if index != -1 && v.ID == "" { // 有座位是空的就把座位索引存起來
			index = i
		}
	}

	if index == -1 { // 沒有找到座位代表房間滿人
		log.Errorf("%s PlayerJoin failed, room is full", logger.LOG_Room)
		return false
	}
	// 設定連線
	g.players[index].Conn_TCP = conn
	g.players[index].Encoder = encoder
	return true
}

// 玩家離開房間
func (g *Room) PlayerLeave(conn net.Conn) {
	g.MutexLock.Lock()
	defer g.MutexLock.Unlock()
	seatIndex := g.getPlayerIndex(conn) // 取得座位索引
	if seatIndex < 0 {
		log.Errorf("%s PlayerLeave failed, get player seat failed", logger.LOG_Room)
		return
	}
	g.players[seatIndex].CloseConnection()
	g.UpdatePlayerStatus()
}

func (g *Room) HandleMessage(conn net.Conn, packet packet.Pack, stop chan struct{}) error {
	seatIndex := g.getPlayerIndex(conn)
	if seatIndex == -1 {
		log.Errorf("%s HandleMessage fialed, Player is not in connection list", logger.LOG_Room)
		return errors.New("HandleMessage fialed, Player is not in connection list")
	}
	g.MutexLock.Lock()
	defer g.MutexLock.Unlock()
	conn.SetDeadline(time.Time{})
	g.players[seatIndex].TimeoutTime = time.Time{}

	// 處理各類型封包
	switch packet.CMD {
	case "CMD類型":
	}
	return nil
}

// 取得玩家座位索引
func (g *Room) getPlayerIndex(conn net.Conn) int {
	for i, v := range g.players {
		if v.Conn_TCP == conn {
			return i
		}
	}
	return -1
}

func (g *Room) StartRun(stop chan struct{}, endGame chan struct{}) {
	go g.gameUpdate(stop, endGame)
	go g.UpdatePlayerLeaveTime(stop)
	go g.StuckCheck(stop)
}

func (g *Room) StuckCheck(stop chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s StuckCheck error: %v.\n%s", logger.LOG_Room, err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	timer := time.NewTicker(15 * time.Second)
	g.lastChangeStateTime = time.Now()
	for {
		<-timer.C
		elapsed := time.Since(g.lastChangeStateTime)
		if (elapsed.Minutes()) >= 3 && g.gameState != End {
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			g.WriteGameErrorLog("StuckCheck")
			stop <- struct{}{}
		}
	}
}

func (r *Room) gameUpdate(stop chan struct{}, endGame chan struct{}) {
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
		if v.Conn_TCP == nil {
			continue
		}
		err := packet.SendPack(v.Encoder, pack)
		if err != nil {
			log.Errorf("%s BroadCastPacket with error: %v", logger.LOG_Room, err)
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
	if r.players[pIndex].Conn_TCP == nil {
		return
	}
	err := packet.SendPack(r.players[pIndex].Encoder, pack)
	if err != nil {
		log.Errorf("%s SendPacketToPlayer error: %v", logger.LOG_Room, err)
		r.players[pIndex].CloseConnection()
		r.UpdatePlayerStatus()
	}
}

// 取得遊戲房中所有玩家狀態
func (r *Room) GetPlayerStatus() [PLAYER_NUMBER]PlayerStatus {
	playerStatuss := [PLAYER_NUMBER]PlayerStatus{}
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
		Content: &packet.UpdateRoomContent{
			PlayerStatuss: r.GetPlayerStatus(),
		},
	})
}

func (g *Room) UpdatePlayerLeaveTime(stop chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("UpdatePlayerLeaveTime error: %v.\n%s", err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	ticker := time.NewTicker(time.Duration(200) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			g.MutexLock.Lock()
			g.PassSecs += UPDATE_INTERVAL_MS / 1000
			for i := 0; i < PLAYER_NUMBER; i++ {
				if g.players[i].Conn_TCP == nil && !g.players[i].IsAI {
					if g.players[i].leaveTimer < MAX_ALLOW_DISCONNECT_SECS {
						g.players[i].leaveTimer += UPDATE_INTERVAL_MS / 1000
					} else {
						g.players[i].leaveTimer = MAX_ALLOW_DISCONNECT_SECS
					}
				} else {
					g.players[i].leaveTimer = 0
				}
			}
			g.MutexLock.Unlock()
		case <-stop:
			return
		}
	}
}

func (g *Room) CheckActionValid(playerIndex int, actionContent PlayerAction) (bool, error) {
	if playerIndex != actionContent.PlayerIndex {
		//log.Printf("Error Player index not the same.")
		return false, errors.New("Error Player index not the same.")
	}
	//先判斷封包ID是不是這次行動
	if g.TellPlayerAction[playerIndex] == nil || g.TellPlayerAction[playerIndex].ActionInfo.ID != actionContent.ID {
		if EnvironmentVersion != "Release" {
			log.Printf("ActionInfo.ID not the same.")
		}
		return false, nil
	}

	if (g.TellPlayerAction[playerIndex].ActionInfo.CanActions & actionContent.ActionType) != actionContent.ActionType {
		fmt.Println("ActionERROR: ", g.TellPlayerAction[playerIndex].ActionInfo.CanActions, actionContent.ActionType)
		g.sendPacketToPlayer(playerIndex, &packet.Pack{
			Command:  "ACTION_ERROR",
			PacketID: -1,
			Content:  nil,
		})
		return false, errors.New("ACTION_ERROR")
	}
	//要打一張牌的情況
	switch actionContent.ActionType {
	case NoneAction:
		if g.TellPlayerAction[playerIndex].ActionInfo.CanActions != NoneAction {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			return false, errors.New("ACTION_ERROR")
		}
	case Discard:
		if !IntArrayContain(g.TellPlayerAction[playerIndex].CanDiscardList, actionContent.TileID) {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			g.WriteGameErrorLog(ERROR_ACTION, map[string]interface{}{
				"ActionType":      actionContent.ActionType,
				"CanDiscardList":  g.TellPlayerAction[playerIndex].CanDiscardList,
				"ActionTile":      actionContent.TileID,
				"CreateTime":      time.Now(),
				"TellCanAction":   g.TellPlayerAction[playerIndex].ActionInfo.CanActions,
				"TellActionTile":  g.TellPlayerAction[playerIndex].ActionInfo.ActionTileID,
				"PlayingRoomName": g.FirebaseDocID,
				"RoundLen":        len(g.jamData.roundDataList),
				"PlayerIndex":     playerIndex,
				"RoundStep":       len(g.roundData.GetActionLog()),
			})
			return false, errors.New("ACTION_ERROR")
		}
	case EarthListen:
		fallthrough
	case NotifyListen:
		isInKey := false
		for key, _ := range g.TellPlayerAction[playerIndex].CanListenList {
			if key == actionContent.TileID {
				isInKey = true
				break
			}
		}
		if !isInKey {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			canListenTiles := make([]int, 0, len(g.TellPlayerAction[playerIndex].CanListenList))
			for key, _ := range g.TellPlayerAction[playerIndex].CanListenList {
				canListenTiles = append(canListenTiles, key)
			}
			g.WriteGameErrorLog(ERROR_ACTION, map[string]interface{}{
				"ActionType":        actionContent.ActionType,
				"CanDiscardList":    g.TellPlayerAction[playerIndex].CanDiscardList,
				"ActionTile":        actionContent.TileID,
				"CanListenTileList": canListenTiles,
				"CreateTime":        time.Now(),
			})
			return false, errors.New("ACTION_ERROR")
		}
	case Chow:
		inList := false
		for _, meldList := range g.TellPlayerAction[playerIndex].CanChowList {
			if IntArrayCompare(meldList.Tiles, actionContent.MeldTiles) {
				inList = true
				break
			}
		}
		if !inList {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			g.WriteGameErrorLog(ERROR_ACTION, map[string]interface{}{
				"ActionType":      actionContent.ActionType,
				"CanChowList":     g.TellPlayerAction[playerIndex].CanChowList,
				"ActionMeldTiles": actionContent.MeldTiles,
				"CreateTime":      time.Now(),
			})
			return false, errors.New("ACTION_ERROR")
		}
	case Pong:
		if !IntArrayCompare(g.TellPlayerAction[playerIndex].CanPongList.Tiles, actionContent.MeldTiles) {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			g.WriteGameErrorLog(ERROR_ACTION, map[string]interface{}{
				"ActionType":      actionContent.ActionType,
				"CanChowList":     g.TellPlayerAction[playerIndex].CanPongList,
				"ActionMeldTiles": actionContent.MeldTiles,
				"CreateTime":      time.Now(),
			})
			return false, errors.New("ACTION_ERROR")
		}
	case ExposedKong:
		fallthrough
	case AddKong:
		inList := false
		for _, meldList := range g.TellPlayerAction[playerIndex].CanKongList {
			if IntArrayCompare(meldList.Tiles, actionContent.MeldTiles) {
				inList = true
				break
			}
		}
		if !inList {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			g.WriteGameErrorLog(ERROR_ACTION, map[string]interface{}{
				"ActionType":      actionContent.ActionType,
				"CanChowList":     g.TellPlayerAction[playerIndex].CanKongList,
				"ActionMeldTiles": actionContent.MeldTiles,
				"CreateTime":      time.Now(),
			})
			return false, errors.New("ACTION_ERROR")
		}
	case ConcealedKong:
		inList := false
		for _, meldList := range g.TellPlayerAction[playerIndex].CanConcealedKongList {
			if IntArrayCompare(meldList.Tiles, actionContent.MeldTiles) {
				inList = true
				break
			}
		}
		if !inList {
			g.sendPacketToPlayer(playerIndex, &packet.Pack{
				Command:  "ACTION_ERROR",
				PacketID: -1,
				Content:  nil,
			})
			g.WriteGameErrorLog(ERROR_ACTION, map[string]interface{}{
				"ActionType":      actionContent.ActionType,
				"CanChowList":     g.TellPlayerAction[playerIndex].CanConcealedKongList,
				"ActionMeldTiles": actionContent.MeldTiles,
				"CreateTime":      time.Now(),
			})
			return false, errors.New("ACTION_ERROR")
		}
	}
	return true, nil
}

// 萬筒條字
func (g *Room) GetWinPointRate(tileType int) int {
	rate := g.GetAllWinPointRate()
	return rate[tileType]
}

func (g *Room) GetAllWinPointRate() [5]int {
	rate := [5]int{1, 1, 1, 1, 1}
	bagBallNum := [5]int{0, 0, 0, 0, 0}
	pachinkoData := g.roundData.Pachinko.GetRevisedPachinkoData()
	allBagBallNum := pachinkoData[1]

	fmt.Println("GetAllWinPointRateStart: ", bagBallNum, pachinkoData, allBagBallNum)

	pushRecord := []int{}
	for i := 0; i < pachinkoData[1]; i++ {
		bag := g.roundData.Pachinko.GoalSlotRecord[i%len(g.roundData.Pachinko.GoalSlotRecord)]
		bagBallNum[bag]++
		pushRecord = append(pushRecord, bag)
	}

	allFull := true
	for i := 1; i < len(rate); i++ { // 0是未進洞
		if bagBallNum[i] >= g.DBmap.FullBagBalls {
			rate[i] = g.roundData.Pachinko.FullRewardRate[i]
		} else {
			allFull = false
		}
	}

	if allFull {
		allFullRate := g.roundData.Pachinko.AllFullRewardRate
		rate = [5]int{allFullRate, allFullRate, allFullRate, allFullRate, allFullRate}
		fmt.Println("GetAllWinPointRateByData: ", pachinkoData)
		fmt.Println("GetAllWinPointRateEnd: (AllFull)BagBallNum:", bagBallNum, " ,pachinkoData:", pachinkoData, " ,pushRecord:", pushRecord)
		fmt.Println("Rate => ", rate)
		return rate
	}

	fmt.Println("GetAllWinPointRateEnd: BagBallNum:", bagBallNum, " ,pachinkoData:", pachinkoData, " ,pushRecord:", pushRecord)
	fmt.Println("GetAllWinPointRateByData: ", pachinkoData)
	fmt.Println("Rate => ", rate, "(", g.roundData.Pachinko.FullRewardRate, ")")
	return rate
}
