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

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	gSetting "matchgame/setting"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

type GameState int // 目前遊戲狀態列舉

const (
	Init GameState = iota
	Start
	End
)

const (
	KICK_PLAYER_SECS     float64 = 60  // 最長允許玩家無心跳X秒後踢出遊戲房
	ATTACK_EXPIRED_SECS  float64 = 10  // 攻擊事件實例被創建後X秒後過期(過期代表再次收到同樣的AttackID時Server不會處理)
	UPDATETIMER_MILISECS int     = 500 // 計時器X毫秒跑一次
)

type Room struct {
	// 玩家陣列(索引0~3 分別代表4個玩家)
	// 1. 索引就是玩家的座位, 一進房間後就不會更動 所以HeroIDs[0]就是在座位0玩家的英雄ID
	// 2. 座位無關玩家進來順序 有人離開就會空著 例如 索引2的玩家離開 Players[2]就會是nil 直到有新玩家加入
	Players      [setting.PLAYER_NUMBER]*Player // 玩家陣列
	RoomName     string                         // 房間名稱(也是DB文件ID)(房主UID+時間轉 MD5)
	GameState    GameState                      // 遊戲狀態
	DBMatchgame  *mongo.DBMatchgame             // DB遊戲房資料
	DBmap        *mongo.DBMap                   // DB地圖設定
	GameTime     float64                        // 遊戲開始X秒
	ErrorLogs    []string                       // ErrorLogs
	MathModel    *gamemath.Model                // 數學模型
	MSpawner     *MonsterSpawner                // 生怪器
	AttackEvents map[string]*AttackEvent        // 攻擊事件
	MutexLock    sync.Mutex
}

// 攻擊事件(包含普攻, 英雄技能, 道具技能, 互動物件等任何攻擊)
// 攻擊事件一段時間清空並存到資料庫中
type AttackEvent struct {
	// 攻擊AttackID格式為 [玩家房間index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	AttackID    string  // 攻擊ID
	ExpiredTime float64 // 過期時間, 房間中的GameTime超過此值就會視為此技能已經結束
	MonsterIdxs [][]int // [波次]-[擊中怪物索引清單]
}

const CHAN_BUFFER = 4

var Env string                       // 環境版本
var MyRoom *Room                     // 房間
var UPDATE_INTERVAL_MS float64 = 100 // 每X毫秒更新一次

func InitGameRoom(dbMapID string, playerIDs [setting.PLAYER_NUMBER]string, roomName string, ip string, port int32, podName string, nodeName string, matchmakerPodName string, roomChan chan *Room) {
	log.Infof("%s InitGameRoom開始", logger.LOG_Room)
	if MyRoom != nil {
		log.Errorf("%s MyRoom已經被初始化過", logger.LOG_Room)
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
		MyRoom.RemoveExpiredAttackEvents() // 移除過期的攻擊事件
	}
}

// 移除過期的攻擊事件
func (r *Room) RemoveExpiredAttackEvents() {
	toRemoveKeys := make([]string, 0)
	for k, v := range r.AttackEvents {
		if r.GameTime > v.ExpiredTime {
			toRemoveKeys = append(toRemoveKeys, k)
		}
	}
	if len(toRemoveKeys) > 0 {
		utility.RemoveFromMapByKeys(r.AttackEvents, toRemoveKeys)
		log.Infof("%s 移除不需要的攻擊事件: %v", logger.LOG_Room, toRemoveKeys)
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
	heroEXP := 0
	heroJson, err := gameJson.GetHeroByID(strconv.Itoa(heroID))
	if err != nil {
		log.Errorf("%s gameJson.GetHeroByID(strconv.Itoa(heroID))", logger.LOG_Room)
		return
	}
	spellJsons := heroJson.GetSpellJsons()
	heroSpells := [3]*HeroSpell{}
	for i := 0; i < 3; i++ {
		heroSpells[i] = &HeroSpell{
			Charge:    0,
			SpellJson: spellJsons[i],
		}
	}
	if player.MyHero != nil {
		heroEXP = player.MyHero.EXP
	}
	player.MyHero = &Hero{
		ID:     heroID,
		SkinID: heroSkinID,
		EXP:    heroEXP,
		Spells: heroSpells,
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
			{Key: "point", Value: player.DBPlayer.Point},     // 設定玩家點數
			{Key: "heroExp", Value: player.DBPlayer.HeroExp}, // 設定英雄經驗
			{Key: "leftGameAt", Value: time.Now()},           // 設定離開遊戲時間
			{Key: "inMatchgameID", Value: ""},                // 設定玩家不在遊戲房內了
			{Key: "redisSync", Value: true},                  // 設定redisSync為true, 代表已經把這次遊玩結果更新上monogoDB了
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
	// 處理各類型封包
	switch pack.CMD {

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

	// ==========擊中怪物==========
	case packet.HIT:
		content := packet.Hit{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleHit(conn, pack, content)
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
	log.Infof("廣播封包給其他玩家 CMD: %v", pack.CMD)
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
			ID:         v.DBPlayer.ID,
			Idx:        v.Index,
			GainPoints: v.GainPoint,
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
			r.GameTime += UPDATE_INTERVAL_MS / 1000 // 更新遊戲時間
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

// 處理收到的攻擊事件(UDP)
func (room *Room) HandleAttack(player *Player, pack packet.UDPReceivePack, content packet.Attack) {
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
	needPoint := int64(room.DBmap.Bet)
	// 取技能表
	spellJson, err := gameJson.GetHeroSpellByID(content.SpellJsonID)
	if err != nil {
		log.Errorf("%s gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
		return
	}
	// 取rtp
	rtp := spellJson.RTP
	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
	// 是否為合法攻擊檢查
	if isSpellAttack { // 如果是技能攻擊, 檢查是否可以施放該技能
		spellIdx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引 Ex.1就是第1個技能
		if err != nil {
			log.Errorf("%s 取施法技能索引錯誤: %v", logger.LOG_Room, err)
		}
		if player.MyHero.CheckCanSpell(spellIdx) {
			log.Errorf("%s 該玩家充能不足, 無法使用技能才對", logger.LOG_Room)
			return
		}
	} else { // 如果是普攻, 檢查是否有足夠點數
		if player.DBPlayer.Point < needPoint {
			log.Errorf("%s 該玩家點數不足, 無法普攻才對", logger.LOG_Room)
			return
		}
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

// // 處理收到的攻擊事件(TCP方法先保留, 未來需要可以改回去)
// func (room *Room) HandleAttack_TCP(conn net.Conn, pack packet.Pack, content packet.Attack) {
// 	// 取玩家
// 	player := room.GetPlayerByTCPConn(conn)
// 	if player == nil {
// 		log.Errorf("%s room.getPlayer為nil", logger.LOG_Room)
// 		return
// 	}
// 	needPoint := int64(room.DBmap.Bet)
// 	// 取技能表
// 	spellJson, err := gameJson.GetHeroSpellByID(content.SpellJsonID)
// 	if err != nil {
// 		log.Errorf("%s gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
// 		return
// 	}
// 	// 取rtp
// 	rtp := spellJson.RTP
// 	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
// 	// 是否為合法攻擊檢查
// 	if isSpellAttack { // 如果是技能攻擊, 檢查是否可以施放該技能
// 		spellIdx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引 Ex.1就是第1個技能
// 		if err != nil {
// 			log.Errorf("%s 取施法技能索引錯誤: %v", logger.LOG_Room, err)
// 		}
// 		if player.MyHero.CheckCanSpell(spellIdx) {
// 			log.Errorf("%s 該玩家充能不足, 無法使用技能才對", logger.LOG_Room)
// 			return
// 		}
// 	} else { // 如果是普攻, 檢查是否有足夠點數
// 		if player.DBPlayer.Point < needPoint {
// 			log.Errorf("%s 該玩家點數不足, 無法普攻才對", logger.LOG_Room)
// 			return
// 		}
// 	}
// 	// 廣播給client
// 	room.BroadCastPacket(&packet.Pack{
// 		CMD:    packet.ATTACK_TOCLIENT,
// 		PackID: pack.PackID,
// 		Content: &packet.Attack_ToClient{
// 			PlayerIdx:   player.Index,
// 			SpellJsonID: content.SpellJsonID,
// 			MonsterIdx:  content.MonsterIdx,
// 		}},
// 	)
// }

// 處理收到的擊中事件
func (room *Room) HandleHit(conn net.Conn, pack packet.Pack, hitCMD packet.Hit) {
	// 取玩家
	player := room.GetPlayerByTCPConn(conn)
	if player == nil {
		log.Errorf("%s room.getPlayer為nil", logger.LOG_Room)
		return
	}
	// 取技能表
	spellJson, err := gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)
	if err != nil {
		log.Errorf("%s gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
		return
	}
	// 取rtp
	rtp := spellJson.RTP
	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
	// 如果是技能攻擊, 檢查是否可以施放該技能
	if isSpellAttack {
		spellIdx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引 Ex.1就是第1個技能
		if err != nil {
			log.Errorf("%s 取施法技能索引錯誤: %v", logger.LOG_Room, err)
		}
		if player.MyHero.CheckCanSpell(spellIdx) {
			log.Errorf("%s 該玩家充能不足, 無法使用技能才對", logger.LOG_Room)
			return
		}
	}
	// 取波次命中數
	spellMaxHits := spellJson.MaxHits
	// 花費點數
	spendPoint := int64(0)
	// 攻擊ID格式為 [玩家index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	attackID := strconv.Itoa(player.Index) + "_" + strconv.Itoa(hitCMD.AttackID)

	hitMonsterIdxs := make([]int, 0)   // 擊中怪物索引清單
	killMonsterIdxs := make([]int, 0)  // 擊殺怪物索引清單, [1,1,3]就是依次擊殺索引為1,1與3的怪物
	gainPoints := make([]int64, 0)     // 獲得點數清單, [1,1,3]就是依次獲得點數1,1與3
	gainSpellCharges := make([]int, 0) // 獲得技能充能清單, [1,1,3]就是依次獲得技能1,技能1,技能3的充能
	gainHeroExps := make([]int, 0)     // 獲得英雄經驗清單, [1,1,3]就是依次獲得英雄經驗1,1與3
	gainDrops := make([]int, 0)        // 獲得掉落清單, [1,1,3]就是依次獲得DropJson中ID為1,1與3的掉落

	// 遍歷擊中的怪物並計算擊殺與獎勵
	hitCMD.MonsterIdxs = utility.RemoveDuplicatesFromSlice(hitCMD.MonsterIdxs) // 移除重複的命中索引
	for _, monsterIdx := range hitCMD.MonsterIdxs {
		// 確認怪物索引存在清單中, 不存在代表已死亡或是client送錯怪物索引
		if monster, ok := room.MSpawner.Monsters[monsterIdx]; ok {

			if monster == nil {
				log.Errorf("%s room.MSpawner.Monsters中的monster is null", logger.LOG_Room)
				continue
			}

			hitMonsterIdxs = append(hitMonsterIdxs, monsterIdx) // 加入擊中怪物索引清單

			// 取得怪物賠率
			odds, err := strconv.ParseFloat(monster.MonsterJson.Odds, 64)
			if err != nil {
				log.Errorf("%s strconv.ParseFloat(monster.MonsterJson.Odds, 64)錯誤: %v", logger.LOG_Room, err)
				return
			}
			// 取得怪物經驗
			monsterExp, err := strconv.ParseFloat(monster.MonsterJson.EXP, 64)
			if err != nil {
				log.Errorf("%s strconv.ParseFloat(monster.MonsterJson.EXP, 64)錯誤: %v", logger.LOG_Room, err)
				return
			}

			// 取得怪物掉落道具
			dropJsons := monster.MonsterJson.GetDropJsonDatas()
			dropAddOdds := 0.0 // 掉落道具增加的總RTP
			for _, v := range dropJsons {
				addOdds, err := strconv.ParseFloat(v.GainRTP, 64)
				if err != nil {
					log.Errorf("%s drop表(ID : %s)的GainRTP轉float錯誤: %v", logger.LOG_Room, v.ID, err)
					continue
				}
				dropID, err := strconv.ParseInt(v.ID, 10, 32)
				if err != nil {
					log.Errorf("%s drop表(ID : %s)的ID轉int錯誤: %v", logger.LOG_Room, v.ID, err)
					continue
				}
				dropAddOdds += addOdds
				gainDrops = append(gainDrops, int(dropID))
			}

			// 計算實際怪物死掉獲得點數
			rewardPoint := int64((odds + dropAddOdds) * float64(room.DBmap.Bet))

			// 計算是否造成擊殺
			kill := false
			rndUnchargedSpell := player.MyHero.GetRandomUnchargedSpell()
			if !isSpellAttack { // 普攻
				// 擊殺判定
				attackKP := room.MathModel.GetAttackKP(odds, int(spellMaxHits), rndUnchargedSpell != nil)
				kill = utility.GetProbResult(attackKP)
				log.Infof("======attackID: %s, spellMaxHits:%v odds:%v attackKP:%v kill:%v ", attackID, spellMaxHits, odds, attackKP, kill)
			} else { // 技能攻擊
				attackKP := room.MathModel.GetSpellKP(rtp, odds, int(spellMaxHits))
				kill = utility.GetProbResult(attackKP)
				log.Infof("======attackID: %s, spellMaxHits:%v rtp: %v odds:%v attackKP:%v kill:%v", attackID, spellMaxHits, rtp, odds, attackKP, kill)
			}

			// 如果有擊殺就加到清單中
			if kill {
				// 技能充能掉落
				dropChargeP := 0.0
				if rndUnchargedSpell != nil {
					dropChargeP = room.MathModel.GetHeroSpellDropP_AttackKilling(rndUnchargedSpell.SpellJson.RTP, odds)
					if utility.GetProbResult(dropChargeP) {
						dropSpellIdx, err := utility.ExtractLastDigit(rndUnchargedSpell.SpellJson.ID) // 掉落充能的技能索引 Ex.1就是第1個技能
						if err != nil {
							log.Errorf("%s utility.ExtractLastDigit(rndUnchargedSpell.SpellJson.ID)錯誤: %v", logger.LOG_Room, err)
						}
						gainSpellCharges = append(gainSpellCharges, dropSpellIdx)
					}
				}
				killMonsterIdxs = append(killMonsterIdxs, monsterIdx)
				gainPoints = append(gainPoints, rewardPoint)
				gainHeroExps = append(gainHeroExps, int(monsterExp))
			}
		}
	}

	// 此波攻擊沒命中任何怪物
	if len(hitMonsterIdxs) == 0 {
		return
	}

	// 以attackID來建立攻擊事件, 如果攻擊事件已存在代表是同一個技能但不同波次的攻擊, 此時就追加擊中怪物清單在該攻擊事件
	if _, ok := room.AttackEvents[attackID]; !ok {
		// 檢查擊中怪物數量是否超過此技能的最大擊中數量
		if len(hitMonsterIdxs) > int(spellMaxHits) {
			log.Errorf("%s 收到的擊中數量超過此技能最大擊中數量: %v", logger.LOG_Room, err)
			return
		}
		idxs := make([][]int, 1)
		idxs[0] = hitMonsterIdxs
		attackEvent := AttackEvent{
			AttackID:    attackID,
			ExpiredTime: room.GameTime + ATTACK_EXPIRED_SECS,
			MonsterIdxs: idxs,
		}
		room.AttackEvents[attackID] = &attackEvent
		// 普攻的話要扣點數
		if !isSpellAttack {
			spendPoint = -int64(room.DBmap.Bet)
		}
	} else {
		attackEvent := room.AttackEvents[attackID]
		if attackEvent == nil {
			log.Errorf("%s room.AttackEvents[attackID]為nil", logger.LOG_Room)
			return
		}
		// 目前此技能收到的總擊中數量
		curHits := len(hitMonsterIdxs)
		for _, innerSlice := range attackEvent.MonsterIdxs {
			curHits += len(innerSlice)
		}
		// 檢查擊中數量是否超過此技能的最大擊中數量
		if curHits > int(spellMaxHits) {
			log.Errorf("%s 收到的擊中數量超過此技能最大可擊中數量: %v", logger.LOG_Room, err)
			return
		} else if curHits == int(spellMaxHits) { // 如果是此攻擊的最後一波命中就移除此Attack
			delete(room.AttackEvents, attackID)
		} else { // 將此波命中加入攻擊中
			attackEvent.MonsterIdxs = append(attackEvent.MonsterIdxs, hitCMD.MonsterIdxs)
		}
	}

	// 玩家點數變化
	totalGainPoint := spendPoint + utility.SliceSum(gainPoints) // 總點數變化是消耗點數+獲得點數
	if totalGainPoint != 0 {
		player.AddPoint(totalGainPoint)
	}
	// 英雄增加經驗
	player.AddHeroExp(utility.SliceSum(gainHeroExps))
	// 英雄技能充能
	for _, v := range gainSpellCharges {
		player.MyHero.AddHeroSpellCharge(v, 1)
	}

	// 從怪物清單中移除被擊殺的怪物
	room.MSpawner.RemoveMonsters(killMonsterIdxs)

	// log.Infof("killMonsterIdxs: %v \n", killMonsterIdxs)
	// log.Infof("gainPoints: %v \n", gainPoints)
	// log.Infof("gainSpellCharges: %v \n", gainSpellCharges)
	// 廣播給client
	room.BroadCastPacket(-1, &packet.Pack{
		CMD:    packet.HIT_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.Hit_ToClient{
			KillMonsterIdxs:  killMonsterIdxs,
			GainPoints:       gainPoints,
			GainHeroExps:     gainHeroExps,
			GainSpellCharges: gainSpellCharges,
			GainDrops:        gainDrops,
		}},
	)
}
