package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"herofishingGoModule/setting"
	"herofishingGoModule/utility"
	"matchgame/gamemath"
	logger "matchgame/logger"
	"matchgame/packet"

	gSetting "matchgame/setting"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

type GameState int // 目前遊戲狀態列舉

const (
	Init GameState = iota
	Start
	End
)

const (
	KICK_PLAYER_SECS     float64 = 60  // 最長允許玩家無心跳X秒後踢出遊戲房
	ATTACK_EXPIRED_SECS  float64 = 3   // 攻擊事件實例被創建後X秒後過期(過期代表再次收到同樣的AttackID時Server不會處理)
	UPDATETIMER_MILISECS int     = 500 // 計時器X毫秒跑一次
)

type Room struct {
	// 玩家陣列(索引0~3 分別代表4個玩家)
	// 1. 索引就是玩家的座位, 一進房間後就不會更動 所以HeroIDs[0]就是在座位0玩家的英雄ID
	// 2. 座位無關玩家進來順序 有人離開就會空著 例如 索引2的玩家離開 Players[2]就會是nil 直到有新玩家加入
	Players         [setting.PLAYER_NUMBER]*Player // 玩家陣列
	RoomName        string                         // 房間名稱(也是DB文件ID)(房主UID+時間轉 MD5)
	GameState       GameState                      // 遊戲狀態
	DBMatchgame     *mongo.DBMatchgame             // DB遊戲房資料
	DBmap           *mongo.DBMap                   // DB地圖設定
	GameTime        float64                        // 遊戲開始X秒
	ErrorLogs       []string                       // ErrorLogs
	MathModel       *gamemath.Model                // 數學模型
	MSpawner        *MonsterSpawner                // 生怪器
	AttackEvents    map[string]*AttackEvent        // 攻擊事件
	ExpireAttackIDs utility.HashSet                // 已過期的攻擊事件ID清單
	SceneEffects    []packet.SceneEffect           // 場景效果
	MutexLock       sync.Mutex
}

// 攻擊事件(包含普攻, 英雄技能, 道具技能, 互動物件等任何攻擊)
// 攻擊事件一段時間清空並存到資料庫中
type AttackEvent struct {
	// 攻擊AttackID格式為 [玩家房間index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	AttackID    string  // 攻擊ID
	ExpiredTime float64 // 過期時間, 房間中的GameTime超過此值就會視為此技能已經結束
	MonsterIdxs [][]int // [波次]-[擊中怪物索引清單]
	// 是否已經支付該攻擊需要的花費(普攻要花費點數, 技能要花費能量)
	// 如果Client收到Hit但還沒收到Attack就會先標示為false, 等到確實收到Attack並支付費用後才會設為true
	Paid              bool
	Hit_ToClientPacks []packet.Pack // 先收到Hit但還沒收到Attack時就把要返回Client的資料先存起來
}

const CHAN_BUFFER = 4

var Env string   // 環境版本
var MyRoom *Room // 房間

func InitGameRoom(dbMapID string, playerIDs [setting.PLAYER_NUMBER]string, roomName string, ip string, port int32, podName string, nodeName string, matchmakerPodName string, roomChan chan *Room) {
	log.Infof("%s InitGameRoom開始", logger.LOG_Room)
	if MyRoom != nil {
		log.Errorf("%s MyRoom已經被初始化過", logger.LOG_Room)
		return
	}

	// 依據dbMapID從DB中取dbMap設定
	log.Infof("%s 取DBMap資料", logger.LOG_Room)
	var dbMap mongo.DBMap
	err := mongo.GetDocByID(mongo.ColName.Map, dbMapID, &dbMap)
	if err != nil {
		log.Errorf("%s InitGameRoom時取dbmap資料發生錯誤", logger.LOG_Room)
	}
	log.Infof("%s 取DBMap資料成功 DBMapID: %s JsonMapID: %v", logger.LOG_Room, dbMap.ID, dbMap.JsonMapID)

	log.Infof("%s 設定dbMatchgame資料", logger.LOG_Room)
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

	log.Infof("%s 初始化房間設定", logger.LOG_Room)
	// 初始化房間設定
	MyRoom = &Room{
		RoomName:    roomName,
		GameState:   Init,
		DBmap:       &dbMap,
		DBMatchgame: &dbMatchgame,
		GameTime:    0,
		MathModel: &gamemath.Model{
			GameRTP:        dbMap.RTP,            // 遊戲RTP
			SpellSharedRTP: dbMap.SpellSharedRTP, // 攻擊RTP
		},
		ExpireAttackIDs: utility.NewHashSet(),
	}
	log.Infof("%s 初始生怪器", logger.LOG_Room)
	// 初始生怪器
	MyRoom.MSpawner = NewMonsterSpawner()
	MyRoom.MSpawner.InitMonsterSpawner(dbMap.JsonMapID)
	MyRoom.AttackEvents = make(map[string]*AttackEvent)
	go RoomLoop() // 開始房間循環
	// 這裡之後要加房間初始化Log到DB

	log.Infof("%s InitGameRoom完成", logger.LOG_Room)
	roomChan <- MyRoom
}

// 房間循環
func RoomLoop() {
	ticker := time.NewTicker(gSetting.ROOMLOOP_MS * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		MyRoom.RemoveExpiredAttackEvents()  // 移除過期的攻擊事件
		MyRoom.RemoveExpiredSceneEffects()  // 移除過期的場景效果
		MyRoom.RemoveExpiredPlayerBuffers() // 移除過期的玩家Buffer
	}
}

// 移除過期的攻擊事件
func (r *Room) RemoveExpiredAttackEvents() {
	toRemoveKeys := make([]string, 0)
	for k, v := range r.AttackEvents {
		if r.GameTime > v.ExpiredTime {
			toRemoveKeys = append(toRemoveKeys, k)
			r.ExpireAttackIDs.Add(v.AttackID)
		}
	}
	if len(toRemoveKeys) > 0 {
		utility.RemoveFromMapByKeys(r.AttackEvents, toRemoveKeys)
		// log.Infof("%s 移除過期的攻擊事件: %v", logger.LOG_Room, toRemoveKeys)
	}
}

// 移除過期的場景效果
func (r *Room) RemoveExpiredSceneEffects() {
	toRemoveIdxs := make([]int, 0)
	for i, v := range r.SceneEffects {
		if r.GameTime > (v.AtTime + v.Duration) {
			toRemoveIdxs = append(toRemoveIdxs, i)
		}
	}
	if len(toRemoveIdxs) > 0 {
		// for _, v := range toRemoveIdxs {
		// 	log.Infof("%s 移除過期的場景效果: %v", logger.LOG_Room, r.SceneEffects[v].Name)
		// }
		r.SceneEffects = utility.RemoveFromSliceBySlice(r.SceneEffects, toRemoveIdxs)
	}
}

// 移除過期的玩家Buffer
func (r *Room) RemoveExpiredPlayerBuffers() {
	for _, player := range r.Players {
		if player == nil {
			return
		}
		toRemoveIdxs := make([]int, 0)
		for j, buffer := range player.PlayerBuffs {
			if r.GameTime > (buffer.AtTime + buffer.Duration) {
				toRemoveIdxs = append(toRemoveIdxs, j)
			}
		}
		if len(toRemoveIdxs) > 0 {
			// for _, v := range toRemoveIdxs {
			// 	log.Infof("%s 移除過期的玩家Buffer: %v", logger.LOG_Room, player.PlayerBuffs[v].Name)
			// }
			player.PlayerBuffs = utility.RemoveFromSliceBySlice(player.PlayerBuffs, toRemoveIdxs)
		}
	}
}

func (r *Room) WriteGameErrorLog(log string) {
	r.ErrorLogs = append(r.ErrorLogs, log)
}

// 取得房間玩家數
func (r *Room) PlayerCount() int {
	count := 0
	for _, v := range r.Players {
		if v != nil {
			count++
		}
	}
	return count
}

// 設定遊戲房內玩家使用英雄ID
func (r *Room) SetHero(conn net.Conn, heroID int, heroSkinID string) {
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()
	player := r.GetPlayerByTCPConn(conn)
	if player == nil {
		log.Errorf("%s SetHero時player := r.getPlayer(conn)為nil", logger.LOG_Room)
		return
	}

	heroJson, err := gameJson.GetHeroByID(strconv.Itoa(heroID))
	if err != nil {
		log.Errorf("%s gameJson.GetHeroByID(strconv.Itoa(heroID))", logger.LOG_Room)
		return
	}
	spellJsons := heroJson.GetSpellJsons()

	heroEXP := 0
	// spellCharges := [3]int{0, 0, 0}
	if player.MyHero != nil {
		heroEXP = player.MyHero.EXP
	}
	player.MyHero = &Hero{
		ID:     heroID,
		SkinID: heroSkinID,
		EXP:    heroEXP,
		Spells: spellJsons,
	}
}

// 取得房間內所有玩家使用英雄與Skin資料
func (r *Room) GetHeroInfos() ([setting.PLAYER_NUMBER]int, [setting.PLAYER_NUMBER]string) {
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()
	var heroIDs [setting.PLAYER_NUMBER]int
	var heroSkinIDs [setting.PLAYER_NUMBER]string
	for i, player := range r.Players {
		if player == nil {
			heroIDs[i] = 0
			heroSkinIDs[i] = ""
			continue
		}
		heroIDs[i] = player.MyHero.ID
		heroSkinIDs[i] = player.MyHero.SkinID
	}
	return heroIDs, heroSkinIDs
}

// 把玩家加到房間中, 成功時回傳true
func (r *Room) JoinPlayer(player *Player) bool {
	if player == nil {
		log.Errorf("%s JoinPlayer傳入nil Player", logger.LOG_Room)
		return false
	}
	log.Infof("%s 玩家 %s 嘗試加入房間", logger.LOG_Room, player.DBPlayer.ID)

	index := -1
	for i, v := range r.Players {
		if v == nil && index == -1 { // 有座位是空的就把座位索引存起來
			index = i
			break
		}
		if v.DBPlayer.ID == player.DBPlayer.ID { // 如果要加入的玩家ID與目前房間的玩家ID一樣就回傳失敗
			log.Errorf("%s 加入房間失敗, 嘗試加入同樣的玩家: %s.\n", logger.LOG_Room, player.DBPlayer.ID)
			return false
		}
	}
	if index == -1 { // 沒有找到座位代表房間滿人
		log.Errorf("%s 房間已滿", logger.LOG_Room)
		return false
	}
	// 設定玩家
	r.MutexLock.Lock()
	player.Index = index
	r.Players[index] = player
	r.MutexLock.Unlock()
	r.OnRoomPlayerChange()
	log.Infof("%s 玩家%s 已加入房間(%v/%v) 房間資訊: %+v", logger.LOG_Room, player.DBPlayer.ID, r.PlayerCount(), setting.PLAYER_NUMBER, r)
	return true
}

// 將玩家踢出房間
func (r *Room) KickPlayer(conn net.Conn) {
	log.Infof("%s 執行KickPlayer", logger.LOG_Room)
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()
	seatIndex := r.GetPlayerIndexByTCPConn(conn) // 取得座位索引
	if seatIndex < 0 || r.Players[seatIndex] == nil {
		return
	}
	player := r.Players[seatIndex]
	// 更新玩家DB
	if player.DBPlayer != nil {
		log.Infof("%s 嘗試踢出玩家 %s", logger.LOG_Room, player.DBPlayer.ID)
		// 更新玩家DB資料
		updatePlayerBson := bson.D{
			{Key: "point", Value: player.DBPlayer.Point},               // 設定玩家點數
			{Key: "leftGameAt", Value: time.Now()},                     // 設定離開遊戲時間
			{Key: "inMatchgameID", Value: ""},                          // 設定玩家不在遊戲房內了
			{Key: "redisSync", Value: true},                            // 設定redisSync為true, 代表已經把這次遊玩結果更新上monogoDB了
			{Key: "heroExp", Value: player.DBPlayer.HeroExp},           // 設定英雄經驗
			{Key: "spellCharges", Value: player.DBPlayer.SpellCharges}, // 設定技能充能
			{Key: "drops", Value: player.DBPlayer.Drops},               // 設定掉落道具
		}
		r.PubPlayerLeftMsg(player.DBPlayer.ID) // 送玩家離開訊息給Matchmaker
		mongo.UpdateDocByID(mongo.ColName.Player, player.DBPlayer.ID, updatePlayerBson)
		log.Infof("%s 更新玩家 %s DB資料玩家", logger.LOG_Room, player.DBPlayer.ID)
	}
	player.RedisPlayer.ClosePlayer() // 關閉該玩家的RedisDB
	player.CloseConnection()
	r.Players[seatIndex] = nil
	r.OnRoomPlayerChange()
	// 更新玩家狀態
	r.BroadCastPacket(seatIndex, &packet.Pack{
		CMD:    packet.UPDATEPLAYER_TOCLIENT,
		PackID: -1,
		Content: &packet.UpdatePlayer_ToClient{
			Players: r.GetPacketPlayers(),
		},
	})
	log.Infof("%s 踢出玩家完成", logger.LOG_Room)
}

// 送玩家離開訊息給Matchmaker
func (r *Room) PubPlayerLeftMsg(playerID string) {
	publishChannelName := "Game-" + r.RoomName
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
	log.Infof("%s 送完加離開訊息到 %s Msg: %+v", logger.LOG_Room, publishChannelName, msg)
}

// 房間人數有異動處理
func (r *Room) OnRoomPlayerChange() {
	if r == nil {
		return
	}
	// 不是空房間處理
	if r.PlayerCount() != 0 {
		r.MSpawner.SpawnSwitch(true) // 開始生怪
		return
	}
	// 如果是空房間處理
	r.MSpawner.SpawnSwitch(false) // 停止生怪

}

// 處理TCP訊息
func (r *Room) HandleTCPMsg(conn net.Conn, pack packet.Pack) error {
	seatIndex := r.GetPlayerIndexByTCPConn(conn)
	if seatIndex == -1 {
		log.Errorf("%s HandleMessage fialed, Player is not in connection list", logger.LOG_Room)
		return errors.New("HandleMessage fialed, Player is not in connection list")
	}
	conn.SetDeadline(time.Time{}) // 移除連線超時設定
	// 取玩家
	player := r.GetPlayerByTCPConn(conn)
	if player == nil {
		log.Errorf("%s room.getPlayer為nil", logger.LOG_Room)
		return fmt.Errorf("%s room.getPlayer為nil, 可能玩家已離開", logger.LOG_Room)
	}
	// 處理各類型封包
	switch pack.CMD {
	// ==========更新場景(玩家剛進遊戲 或 斷線回連會主動跟Server要更新資料用)==========
	case packet.UPDATESCENE:
		r.SendPacketToPlayer(player.Index, &packet.Pack{
			CMD:    packet.UPDATESCENE_TOCLIENT,
			PackID: -1,
			Content: &packet.UpdateScene_ToClient{
				Spawns:       r.MSpawner.Spawns,
				SceneEffects: r.SceneEffects,
			},
		})
	// ==========設定英雄==========
	case packet.SETHERO:
		content := packet.SetHero{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		r.SetHero(conn, content.HeroID, content.HeroSkinID) // 設定使用的英雄ID
		heroIDs, heroSkinIDs := r.GetHeroInfos()
		// 廣播給所有玩家
		r.BroadCastPacket(-1, &packet.Pack{ // 廣播封包
			CMD: packet.SETHERO_TOCLIENT,
			Content: &packet.SetHero_ToClient{
				HeroIDs:     heroIDs,
				HeroSkinIDs: heroSkinIDs,
			},
		})

	// ==========離開遊戲房==========
	case packet.LEAVE: //離開遊戲房
		content := packet.Leave{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		r.KickPlayer(conn) // 將玩家踢出房間
	// ==========發動攻擊==========
	case packet.ATTACK:
		content := packet.Attack{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleAttack(player, pack, content)
	// ==========擊中怪物==========
	case packet.HIT:
		content := packet.Hit{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleHit(player, pack, content)
	// ==========使用道具==========
	case packet.DROPSPELL:
		content := packet.DropSpell{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleDropSpell(player, pack, content)
	}

	return nil
}

// 透過TCPConn取得玩家座位索引
func (r *Room) GetPlayerIndexByTCPConn(conn net.Conn) int {
	for i, v := range r.Players {
		if v == nil || v.ConnTCP == nil {
			continue
		}

		if v.ConnTCP.Conn == conn {
			return i
		}
	}
	return -1
}

// 透過ConnToken取得玩家座位索引
func (r *Room) GetPlayerIndexByConnToken(connToken string) int {
	for i, v := range r.Players {
		if v == nil || v.ConnUDP == nil {
			continue
		}

		if v.ConnUDP.ConnToken == connToken {
			return i
		}
	}
	return -1
}

// 透過TCPConn取得玩家
func (r *Room) GetPlayerByTCPConn(conn net.Conn) *Player {
	for _, v := range r.Players {
		if v == nil || v.ConnTCP == nil {
			continue
		}

		if v.ConnTCP.Conn == conn {
			return v
		}
	}
	return nil
}

// 透過ConnToken取得玩家
func (r *Room) GetPlayerByConnToken(connToken string) *Player {
	for _, v := range r.Players {
		if v == nil || v.ConnUDP == nil {
			continue
		}
		if v.ConnUDP.ConnToken == connToken {
			return v
		}
	}
	return nil
}

// 改變遊戲狀態
func (r *Room) ChangeState(state GameState) {
	r.GameState = state
}

// 送封包給遊戲房間內所有玩家(TCP), 除了指定索引(exceptPlayerIdx)的玩家, 如果要所有玩家就傳入-1就可以
func (r *Room) BroadCastPacket(exceptPlayerIdx int, pack *packet.Pack) {
	if pack.CMD != packet.SPAWN_TOCLIENT {
		log.Infof("廣播封包給其他玩家 CMD: %v", pack.CMD)
	}

	// 送封包給所有房間中的玩家
	for i, v := range r.Players {
		if i == exceptPlayerIdx {
			continue
		}
		if v == nil || v.ConnTCP.Conn == nil {
			continue
		}
		err := packet.SendPack(v.ConnTCP.Encoder, pack)
		if err != nil {
			log.Errorf("%s 廣播封包錯誤: %v", logger.LOG_Room, err)
		}
	}
}

// 送封包給玩家(TCP)
func (r *Room) SendPacketToPlayer(pIndex int, pack *packet.Pack) {
	if r.Players[pIndex] == nil || r.Players[pIndex].ConnTCP.Conn == nil {
		return
	}
	err := packet.SendPack(r.Players[pIndex].ConnTCP.Encoder, pack)
	if err != nil {
		log.Errorf("%s SendPacketToPlayer error: %v", logger.LOG_Room, err)
		r.KickPlayer(r.Players[pIndex].ConnTCP.Conn)
	}
}

// 取得要送封包的玩家陣列
func (r *Room) GetPacketPlayers() [setting.PLAYER_NUMBER]*packet.Player {
	var players [setting.PLAYER_NUMBER]*packet.Player
	for i, v := range r.Players {
		if v == nil {
			players[i] = nil
			continue
		}
		players[i] = &packet.Player{
			ID:          v.DBPlayer.ID,
			Idx:         v.Index,
			GainPoints:  v.GainPoint,
			PlayerBuffs: v.PlayerBuffs,
		}
	}
	return players
}

// 送封包給玩家(UDP)
func (r *Room) SendPacketToPlayer_UDP(pIndex int, sendData []byte) {
	if r.Players[pIndex] == nil || r.Players[pIndex].ConnUDP.Conn == nil {
		return
	}
	if sendData == nil {
		return
	}
	player := r.Players[pIndex]
	sendData = append(sendData, '\n')
	_, sendErr := player.ConnUDP.Conn.WriteTo(sendData, player.ConnUDP.Addr)
	if sendErr != nil {
		log.Errorf("%s (UDP)送封包錯誤 %s", logger.LOG_Main, sendErr.Error())
		return
	}
}

// 送封包給遊戲房間內所有玩家(UDP), 除了指定索引(exceptPlayerIdx)的玩家, 如果要所有玩家就傳入-1就可以
func (r *Room) BroadCastPacket_UDP(exceptPlayerIdx int, sendData []byte) {
	if sendData == nil {
		return
	}
	for i, v := range r.Players {
		if exceptPlayerIdx == i {
			continue
		}
		if v == nil || v.ConnUDP.Conn == nil {
			continue
		}
		sendData = append(sendData, '\n')
		_, sendErr := v.ConnUDP.Conn.WriteTo(sendData, v.ConnUDP.Addr)
		if sendErr != nil {
			log.Errorf("%s (UDP)送封包錯誤 %s", logger.LOG_Main, sendErr.Error())
			return
		}
	}
}

// 遊戲計時器
func (r *Room) RoomTimer(stop chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s UpdateTimer錯誤: %v.\n%s", logger.LOG_Room, err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	ticker := time.NewTicker(time.Duration(UPDATETIMER_MILISECS) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			r.GameTime += float64(UPDATETIMER_MILISECS) / 1000 // 更新遊戲時間
			for _, player := range r.Players {
				if player == nil {
					continue
				}
				nowTime := time.Now()
				// 玩家無心跳超過X秒就踢出遊戲房
				// log.Infof("%s 目前玩家 %s 已經無回應 %.0f 秒了", logger.LOG_Room, player.DBPlayer.ID, nowTime.Sub(player.LastUpdateAt).Seconds())
				if nowTime.Sub(player.LastUpdateAt) > time.Duration(KICK_PLAYER_SECS)*time.Second {
					MyRoom.KickPlayer(player.ConnTCP.Conn)
				}
			}
		case <-stop:
			return
		}
	}
}

// 處理收到的攻擊事件
func (room *Room) HandleAttack(player *Player, pack packet.Pack, content packet.Attack) {

	// 攻擊ID格式為 [玩家index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	attackID := strconv.Itoa(player.Index) + "_" + strconv.Itoa(content.AttackID)
	if room.ExpireAttackIDs.Contains(attackID) { // 此攻擊已經過期
		log.Errorf("%s AttackID: %s 已過期", logger.LOG_Room, attackID)
		return
	}
	// 如果有鎖定目標怪物, 檢查目標怪是否存在, 不存在就返回
	if content.MonsterIdx >= 0 {
		if monster, ok := room.MSpawner.Monsters[content.MonsterIdx]; ok {
			if monster == nil {
				return
			}
		} else {
			return
		}
	}
	// needPoint := int64(room.DBmap.Bet)
	// 取技能表
	spellJson, err := gameJson.GetHeroSpellByID(content.SpellJsonID)
	if err != nil {
		log.Errorf("%s gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
		return
	}
	// 取rtp
	rtp := spellJson.RTP
	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
	spellIdx := 0             // 釋放第幾個技能, 0就代表是普攻
	spendSpellCharge := 0     // 花費技能充能
	spendPoint := int64(0)    // 花費點數

	// 如果是技能攻擊, 設定spellIdx(第幾招技能), 並檢查充能是否足夠
	if isSpellAttack {
		idx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
		spellIdx = idx
		if err != nil {
			room.SendPacketToPlayer(player.Index, &packet.Pack{
				CMD:     packet.HIT_TOCLIENT,
				PackID:  pack.PackID,
				ErrMsg:  "HandleAttack時取技能索引ID錯誤",
				Content: &packet.Hit_ToClient{},
			})
			log.Errorf("%s 取施法技能索引錯誤: %v", logger.LOG_Room, err)
			return
		}
		// 檢查CD
		if spellIdx < 1 || spellIdx > 3 {
			log.Errorf("%s 技能索引不為1~3: %v", logger.LOG_Room, spellIdx)
			return
		}
		passSec := room.GameTime - player.LastSpellsTime[spellIdx-1] // 距離上次攻擊經過的秒數
		if passSec < spellJson.CD {
			log.Errorf("%s 玩家%s的技能仍在CD中, 不應該能施放技能, passSec: %v cd: %v", logger.LOG_Room, player.DBPlayer.ID, passSec, spellJson.CD)
			return
		}
		// 檢查是否可以施放該技能
		if player.CanSpell(spellIdx) {
			log.Errorf("%s 該玩家充能不足, 無法使用技能才對", logger.LOG_Room)
			return
		}
		spell, getSpellErr := player.MyHero.GetSpell(spellIdx)
		if getSpellErr != nil {
			log.Errorf("%s player.MyHero.GetSpell(spellIdx)錯誤: %v", logger.LOG_Room, getSpellErr)
			return
		}

		spendSpellCharge = spell.Cost
		player.LastSpellsTime[spellIdx-1] = room.GameTime

	} else { // 如果是普攻, 檢查是否有足夠點數
		// 檢查CD, 普攻的CD要考慮Buff
		// passSec := room.GameTime - player.LastAttackTime // 距離上次攻擊經過的秒數
		// cd := spellJson.CD / player.GetAttackCDBuff()    // CD秒數
		// if passSec < cd {
		// 	log.Errorf("%s 玩家%s的攻擊仍在CD中, 不應該能攻擊, passSec: %v cd: %v", logger.LOG_Room, player.DBPlayer.ID, passSec, cd)
		// 	return
		// }

		// (先關閉點數不足檢測)
		// 檢查點數
		// if player.DBPlayer.Point < needPoint {
		// 	log.Errorf("%s 該玩家點數不足, 無法普攻才對", logger.LOG_Room)
		// 	return
		// }
		spendPoint = -int64(room.DBmap.Bet)
		player.LastAttackTime = room.GameTime // 設定上一次攻擊時間
	}
	// =============建立攻擊事件=============
	var attackEvent *AttackEvent
	// 以attackID來建立攻擊事件, 如果攻擊事件已存在代表是同一個技能但不同波次的攻擊, 此時就追加擊中怪物清單在該攻擊事件
	if _, ok := room.AttackEvents[attackID]; !ok {
		idxs := make([][]int, 0)
		attackEvent = &AttackEvent{
			AttackID:          attackID,
			ExpiredTime:       room.GameTime + ATTACK_EXPIRED_SECS,
			MonsterIdxs:       idxs,
			Paid:              true,
			Hit_ToClientPacks: make([]packet.Pack, 0),
		}
		room.AttackEvents[attackID] = attackEvent // 將此攻擊事件加入清單
	} else { // 有同樣的攻擊事件存在代表Hit比Attack先送到
		attackEvent = room.AttackEvents[attackID]
		attackEvent.Paid = true // 設為已支付費用
		// 有Hit先送到的封包要處理
		if len(attackEvent.Hit_ToClientPacks) > 0 {
			for _, v := range attackEvent.Hit_ToClientPacks {
				room.settleHit(player, v)
			}
		}
	}

	// =============是合法的攻擊就進行資源消耗與回送封包=============

	// 玩家點數變化
	player.AddPoint(spendPoint)
	// 施放技能的話要減少英雄技能充能
	if spellIdx != 0 && spendSpellCharge != 0 {
		player.AddSpellCharge(spellIdx, -spendSpellCharge)
	}

	// 廣播給client
	room.BroadCastPacket(player.Index, &packet.Pack{
		CMD:    packet.ATTACK_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.Attack_ToClient{
			PlayerIdx:   player.Index,
			SpellJsonID: content.SpellJsonID,
			MonsterIdx:  content.MonsterIdx,
		}},
	)
}

// 處理收到的擊中事件
func (room *Room) HandleHit(player *Player, pack packet.Pack, content packet.Hit) {
	// 攻擊ID格式為 [玩家index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	attackID := strconv.Itoa(player.Index) + "_" + strconv.Itoa(content.AttackID)
	if room.ExpireAttackIDs.Contains(attackID) { // 此攻擊已經過期
		log.Errorf("%s AttackID: %s 已過期", logger.LOG_Room, attackID)
		return
	}

	// 取技能表
	spellJson, err := gameJson.GetHeroSpellByID(content.SpellJsonID)
	if err != nil {
		room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤", pack))
		log.Errorf("%s HandleHit時gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
		return
	}
	// 取rtp
	rtp := spellJson.RTP
	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
	// 取波次命中數
	spellMaxHits := spellJson.MaxHits

	hitMonsterIdxs := make([]int, 0)   // 擊中怪物索引清單
	killMonsterIdxs := make([]int, 0)  // 擊殺怪物索引清單, [1,1,3]就是依次擊殺索引為1,1與3的怪物
	gainPoints := make([]int64, 0)     // 獲得點數清單, [1,1,3]就是依次獲得點數1,1與3
	gainSpellCharges := make([]int, 0) // 獲得技能充能清單, [1,1,3]就是依次獲得技能1,技能1,技能3的充能
	gainHeroExps := make([]int, 0)     // 獲得英雄經驗清單, [1,1,3]就是依次獲得英雄經驗1,1與3
	gainDrops := make([]int, 0)        // 獲得掉落清單, [1,1,3]就是依次獲得DropJson中ID為1,1與3的掉落
	// 遍歷擊中的怪物並計算擊殺與獎勵
	content.MonsterIdxs = utility.RemoveDuplicatesFromSlice(content.MonsterIdxs) // 移除重複的命中索引
	for _, monsterIdx := range content.MonsterIdxs {
		// 確認怪物索引存在清單中, 不存在代表已死亡或是client送錯怪物索引
		if monster, ok := room.MSpawner.Monsters[monsterIdx]; !ok {
			errStr := fmt.Sprintf("目標不存在(或已死亡) monsterIdx:%d", monsterIdx)
			room.SendPacketToPlayer(player.Index, newHitErrorPack(errStr, pack))
			log.Errorf("%s %s", logger.LOG_Room, errStr)
			continue
		} else {
			if monster == nil {
				room.SendPacketToPlayer(player.Index, newHitErrorPack("room.MSpawner.Monsters中的monster is null", pack))
				log.Errorf("%s room.MSpawner.Monsters中的monster is null", logger.LOG_Room)
				continue
			}

			hitMonsterIdxs = append(hitMonsterIdxs, monsterIdx) // 加入擊中怪物索引清單

			// 取得怪物賠率
			odds, err := strconv.ParseFloat(monster.MonsterJson.Odds, 64)
			if err != nil {
				room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取怪物賠率錯誤", pack))
				log.Errorf("%s strconv.ParseFloat(monster.MonsterJson.Odds, 64)錯誤: %v", logger.LOG_Room, err)
				return
			}
			// 取得怪物經驗
			monsterExp, err := strconv.ParseFloat(monster.MonsterJson.EXP, 64)
			if err != nil {
				room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取怪物經驗錯誤", pack))
				log.Errorf("%s strconv.ParseFloat(monster.MonsterJson.EXP, 64)錯誤: %v", logger.LOG_Room, err)
				return
			}

			// 取得怪物掉落道具
			dropAddOdds := 0.0   // 掉落道具增加的總RTP
			dropID64 := int64(0) // 怪物掉落ID
			// 怪物必須有掉落物才需要考慮怪物掉落
			if monster.MonsterJson.DropID != "" {
				dropJson, err := gameJson.GetDropByID(monster.MonsterJson.DropID)
				if err != nil {
					room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取掉落表錯誤", pack))
					log.Errorf("%s HandleHit時gameJson.GetDropByID(monster.MonsterJson.DropID)錯誤: %v", logger.LOG_Room, err)
					return
				}
				dropID64, err = strconv.ParseInt(monster.MonsterJson.DropID, 10, 64)
				if err != nil {
					log.Errorf("%s HandleHit時strconv.ParseInt(monster.MonsterJson.DropID, 10, 64)錯誤: %v", logger.LOG_Room, err)
					return
				}
				// 玩家目前還沒擁有該掉落ID 才需要考慮怪物掉落
				if !player.IsOwnedDrop(int(dropID64)) {
					addOdds, err := strconv.ParseFloat(dropJson.GainRTP, 64)
					if err != nil {
						room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取掉落表的賠率錯誤", pack))
						log.Errorf("%s HandleHit時strconv.ParseFloat(dropJson.GainRTP, 64)錯誤: %v", logger.LOG_Room, err)
						return
					}
					dropAddOdds += addOdds
				}
			}

			// 計算實際怪物死掉獲得點數
			rewardPoint := int64((odds + dropAddOdds) * float64(room.DBmap.Bet))

			// 計算是否造成擊殺
			kill := false
			rndUnchargedSpell, gotUnchargedSpell := player.GetRandomUnchargedSpell()
			if !isSpellAttack { // 普攻
				// 擊殺判定
				attackKP := room.MathModel.GetAttackKP(odds, int(spellMaxHits), gotUnchargedSpell)
				kill = utility.GetProbResult(attackKP)
				// log.Infof("======spellMaxHits:%v odds:%v attackKP:%v kill:%v ", spellMaxHits, odds, attackKP, kill)
			} else { // 技能攻擊
				attackKP := room.MathModel.GetSpellKP(rtp, odds, int(spellMaxHits))
				kill = utility.GetProbResult(attackKP)
				// log.Infof("======spellMaxHits:%v rtp: %v odds:%v attackKP:%v kill:%v", spellMaxHits, rtp, odds, attackKP, kill)
			}

			// 如果有擊殺就加到清單中
			if kill {
				// 技能充能掉落
				dropChargeP := 0.0
				gainSpellCharges = append(gainSpellCharges, -1)
				gainDrops = append(gainDrops, -1)
				if gotUnchargedSpell {
					dropChargeP = room.MathModel.GetHeroSpellDropP_AttackKilling(rndUnchargedSpell.RTP, odds)
					if utility.GetProbResult(dropChargeP) {
						dropSpellIdx, err := utility.ExtractLastDigit(rndUnchargedSpell.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
						if err != nil {
							log.Errorf("%s HandleHit時utility.ExtractLastDigit(rndUnchargedSpell.ID)錯誤: %v", logger.LOG_Room, err)
							room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時解析第X技能索引錯誤", pack))
							return
						}
						gainSpellCharges[len(gainSpellCharges)-1] = dropSpellIdx
					}
				}
				killMonsterIdxs = append(killMonsterIdxs, monsterIdx)
				gainPoints = append(gainPoints, rewardPoint)
				gainHeroExps = append(gainHeroExps, int(monsterExp))
				if dropAddOdds != 0 && dropID64 != 0 {
					gainDrops[len(gainDrops)-1] = int(dropID64)
				}
			}
		}
	}

	// 設定AttackEvent
	var attackEvent *AttackEvent
	// 不存在此攻擊事件代表之前的Attack封包還沒送到
	if _, ok := room.AttackEvents[attackID]; !ok {
		idxs := make([][]int, 1)
		idxs[0] = hitMonsterIdxs
		attackEvent = &AttackEvent{
			AttackID:          attackID,
			ExpiredTime:       room.GameTime + ATTACK_EXPIRED_SECS,
			MonsterIdxs:       idxs,
			Paid:              false, // 設定為還沒支付費用
			Hit_ToClientPacks: make([]packet.Pack, 0),
		}
		room.AttackEvents[attackID] = attackEvent // 將此攻擊事件加入清單

	} else {
		attackEvent = room.AttackEvents[attackID]
		if attackEvent == nil {
			room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時room.AttackEvents[attackID]為nil", pack))
			log.Errorf("%s room.AttackEvents[attackID]為nil", logger.LOG_Room)
			return
		}
		attackEvent.MonsterIdxs = append(attackEvent.MonsterIdxs, hitMonsterIdxs)
	}

	// 計算目前此技能收到的總擊中數量 並檢查 是否超過此技能的最大擊中數量
	hitCount := 0
	for _, innerSlice := range attackEvent.MonsterIdxs {
		hitCount += len(innerSlice)
	}
	if hitCount > int(spellMaxHits) {
		log.Error(content.MonsterIdxs)
		errLog := fmt.Sprintf("HandleHit時收到的擊中數量超過此技能最大可擊中數量, SpellID: %s curHit: %v", spellJson.ID, hitCount)
		log.Error(errLog)
		room.SendPacketToPlayer(player.Index, newHitErrorPack(errLog, pack))

		return
	}
	attackEvent.MonsterIdxs = append(attackEvent.MonsterIdxs, content.MonsterIdxs) // 將此波命中加入攻擊事件中
	// 將命中結果封包計入在此攻擊事件中
	hitPack := packet.Pack{
		CMD:    packet.HIT_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.Hit_ToClient{
			KillMonsterIdxs:  killMonsterIdxs,
			GainPoints:       gainPoints,
			GainHeroExps:     gainHeroExps,
			GainSpellCharges: gainSpellCharges,
			GainDrops:        gainDrops,
		}}
	attackEvent.Hit_ToClientPacks = append(attackEvent.Hit_ToClientPacks, hitPack)
	// log.Errorf("attackEvent.Paid: %v   killMonsterIdxs: %v", attackEvent.Paid, killMonsterIdxs)
	// =============已完成支付費用的命中就進行資源消耗與回送封包=============
	if attackEvent.Paid {
		room.settleHit(player, hitPack)
	}

}

// 已付費的Attack事件才會結算命中
func (room *Room) settleHit(player *Player, hitPack packet.Pack) {

	var content *packet.Hit_ToClient
	if c, ok := hitPack.Content.(*packet.Hit_ToClient); !ok {
		log.Errorf("%s hitPack.Content無法斷言為Hit_ToClient", logger.LOG_Room)
		return
	} else {
		content = c
	}
	// 玩家點數變化
	totalGainPoint := utility.SliceSum(content.GainPoints) // 把 每個擊殺獲得點數加總就是 總獲得點數
	if totalGainPoint != 0 {
		player.AddPoint(totalGainPoint)
	}

	// 英雄增加經驗
	totalGainHeroExps := utility.SliceSum(content.GainHeroExps) // 把 每個擊殺獲得英雄經驗加總就是 總獲得英雄經驗
	player.AddHeroExp(totalGainHeroExps)
	// 擊殺怪物增加英雄技能充能
	for _, v := range content.GainSpellCharges {
		if v <= 0 { // 因為有擊殺但沒掉落充能時, gainSpellCharges仍會填入-1, 所以要加判斷
			continue
		}
		player.AddSpellCharge(v, 1)
	}
	// 擊殺怪物獲得掉落道具
	for _, dropID := range content.GainDrops {
		if dropID <= 0 { // 因為有擊殺但沒掉落時, gainDrops仍會填入-1, 所以要加判斷
			continue
		}
		player.AddDrop(dropID)
	}
	// 從怪物清單中移除被擊殺的怪物(付費後才算目標死亡, 沒收到付費的Attack封包之前都還是算怪物存活)
	room.MSpawner.RemoveMonsters(content.KillMonsterIdxs)
	log.Errorf("killMonsterIdxs: %v gainPoints: %v gainHeroExps: %v gainSpellCharges: %v  , gainDrops: %v ", content.KillMonsterIdxs, content.GainPoints, content.GainHeroExps, content.GainSpellCharges, content.GainDrops)
	// log.Infof("/////////////////////////////////")
	// log.Infof("killMonsterIdxs: %v \n", killMonsterIdxs)
	// log.Infof("gainPoints: %v \n", gainPoints)
	// log.Infof("gainHeroExps: %v \n", gainHeroExps)
	// log.Infof("gainSpellCharges: %v \n", gainSpellCharges)
	// log.Infof("gainDrops: %v \n", gainDrops)
	// 廣播給client
	room.BroadCastPacket(-1, &hitPack)
}

// 處理收到的掉落施法封包(TCP)
func (room *Room) HandleDropSpell(player *Player, pack packet.Pack, content packet.DropSpell) {
	dropSpellJson, err := gameJson.GetDropSpellByID(strconv.Itoa(content.DropSpellJsonID))
	if err != nil {
		log.Errorf("%s HandleDropSpell時gameJson.GetDropSpellByID(strconv.Itoa(content.DropSpellJsonID))錯誤: %v", logger.LOG_Room, err)
		return
	}
	dropSpellID, err := strconv.ParseInt(dropSpellJson.ID, 10, 64)
	if err != nil {
		log.Errorf("%s HandleDropSpell時strconv.ParseInt(dropSpellJson.ID, 10, 64)錯誤: %v", logger.LOG_Room, err)
		return
	}
	ownedDrop := player.IsOwnedDrop(int(dropSpellID))
	if !ownedDrop {
		log.Errorf("%s 玩家%s 無此DropID, 不應該能使用DropSpell: %v", logger.LOG_Room, player.DBPlayer.ID, dropSpellID)
		return
	}
	switch dropSpellJson.EffectType {
	case "Frozen": // 冰風暴
		duration, err := strconv.ParseFloat(dropSpellJson.EffectValue1, 64)
		if err != nil {
			log.Errorf("%s HandleDropSpell時strconv.ParseFloat(dropSpellJson.EffectValue1, 64)錯誤: %v", logger.LOG_Room, err)
			return
		}
		room.SceneEffects = append(room.SceneEffects, packet.SceneEffect{
			Name:     dropSpellJson.EffectType,
			AtTime:   room.GameTime,
			Duration: duration,
		})
		room.BroadCastPacket(player.Index, &packet.Pack{
			CMD:    packet.UPDATESCENE_TOCLIENT,
			PackID: -1,
			Content: &packet.UpdateScene_ToClient{
				Spawns:       room.MSpawner.Spawns,
				SceneEffects: room.SceneEffects,
			},
		})
	case "Speedup": // 急速神符
	default:
		log.Errorf("%s HandleDropSpell傳入尚未定義的EffectType類型: %v", logger.LOG_Room, dropSpellJson.EffectType)
		return
	}
	// 施法後要移除該掉落
	player.RemoveDrop(int(dropSpellID))
}

// 取得hitError封包
func newHitErrorPack(errStr string, pack packet.Pack) *packet.Pack {
	return &packet.Pack{
		CMD:     packet.HIT_TOCLIENT,
		PackID:  pack.PackID,
		ErrMsg:  errStr,
		Content: &packet.Hit_ToClient{},
	}
}
