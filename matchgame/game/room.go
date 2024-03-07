package game

import (
	"errors"
	"fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/setting"
	"herofishingGoModule/utility"

	// "matchgame/agones"
	logger "matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	// agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
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
	TIMELOOP_MILISECS   int     = 100 // 遊戲每X毫秒循環
	KICK_PLAYER_SECS    float64 = 60  // 最長允許玩家無心跳X秒後踢出遊戲房
	ATTACK_EXPIRED_SECS float64 = 5   // 攻擊事件實例被創建後X秒後過期(過期代表再次收到同樣的AttackID時Server不會處理)
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
	MathModel    *MathModel                     // 數學模型
	MSpawner     *MonsterSpawner                // 生怪器
	AttackEvents map[string]*AttackEvent        // 攻擊事件
	SceneEffects []packet.SceneEffect           // 場景效果
	MutexLock    sync.RWMutex
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

var MyRoom *Room // 房間

// Mode模式分為以下:
// standard:一般版本
// non-agones: 個人測試模式(不使用Agones服務, non-agones的連線方式不會透過Matchmaker分配房間再把ip回傳給client, 而是直接讓client去連資料庫matchgame的ip)
var Mode string

func InitGameRoom(dbMapID string, playerIDs [setting.PLAYER_NUMBER]string, roomName string, ip string, port int32, podName string, nodeName string, matchmakerPodName string, roomChan chan *Room) {
	log.Infof("%s InitGameRoom開始", logger.LOG_Room)
	if MyRoom != nil {
		log.Errorf("%s MyRoom已經被初始化過", logger.LOG_Room)
		return
	}
	// 依據dbMapID從DB中取dbMap設定
	log.Infof("%s 取DBMap資料", logger.LOG_Room)
	var dbMap mongo.DBMap
	dbMapErr := mongo.GetDocByID(mongo.ColName.Map, dbMapID, &dbMap)
	if dbMapErr != nil {
		log.Errorf("%s InitGameRoom時取dbmap資料發生錯誤: %v", logger.LOG_Room, dbMapErr)
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
	// 取gameSetting表並設定MatchModel
	log.Infof("%s 取GameSetting資料", logger.LOG_Room)
	var dbGameConfig mongo.DBGameConfig
	dbConfigErr := mongo.GetDocByID(mongo.ColName.GameSetting, "GameConfig", &dbGameConfig)
	if dbConfigErr != nil {
		log.Errorf("%s InitGameRoom時取dbGameConfig資料發生錯誤: %v", logger.LOG_Room, dbConfigErr)
	}
	log.Infof("%s 取DBMap資料成功 DBMapID: %s JsonMapID: %v", logger.LOG_Room, dbMap.ID, dbMap.JsonMapID)
	// 初始化房間設定
	MyRoom = &Room{
		RoomName:    roomName,
		GameState:   Init,
		DBmap:       &dbMap,
		DBMatchgame: &dbMatchgame,
		GameTime:    0,
		MathModel: &MathModel{
			GameRTP:        0.95,                 // 遊戲RTP
			SpellSharedRTP: dbMap.SpellSharedRTP, // 分配給技能掉落的RTP

			// ※RTP校正規則參考英雄捕魚規劃的RTP與期望值計算分頁
			RtpAdjust_KillRateValue: dbGameConfig.RTPAdjust_KillRateValue, // 當玩家實際RTP與遊戲RTP差值大於RTP校正閥值才會進行校正
			RtpAdjust_RTPThreshold:  dbGameConfig.RTPAdjust_RTPThreshold,  // 代表校正時, 擊殺率的改變值
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
		MyRoom.RemoveExpiredAttackEvents()  // 移除過期的攻擊事件
		MyRoom.RemoveExpiredSceneEffects()  // 移除過期的場景效果
		MyRoom.RemoveExpiredPlayerBuffers() // 移除過期的玩家Buffer
	}
}

// 傳入玩家ID取得Player
func (r *Room) GetPlayerByID(playerID string) *Player {
	for _, v := range r.Players {
		if v.DBPlayer != nil && v.DBPlayer.ID == playerID {
			return v
		}
	}
	return nil
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
		// log.Infof("%s 移除過期的攻擊事件: %v", logger.LOG_Room, toRemoveKeys)
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
func (r *Room) SetHero(player *Player, heroID int, heroSkinID string) {
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()

	heroJson, err := gameJson.GetHeroByID(strconv.Itoa(heroID))
	if err != nil {
		log.Errorf("%s gameJson.GetHeroByID(strconv.Itoa(heroID))", logger.LOG_Room)
		return
	}
	spellJsons := heroJson.GetSpellJsons()

	player.MyHero = &Hero{
		ID:     heroID,
		SkinID: heroSkinID,
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
	log.Infof("%s 玩家 %s 嘗試加入房間 DBMatchgame: %+v", logger.LOG_Room, player.DBPlayer.ID, r.DBMatchgame)

	index := -1
	for i, v := range r.Players {
		if v != nil && v.DBPlayer.ID == player.DBPlayer.ID { // 如果要加入的玩家ID與目前房間的玩家ID一樣就回傳失敗
			log.Errorf("%s 加入房間失敗, 嘗試加入同樣的玩家: %s.\n", logger.LOG_Room, player.DBPlayer.ID)
			return false
		}
		if v == nil && index == -1 { // 有座位是空的就把座位索引存起來
			index = i
		}
	}
	if index == -1 { // 沒有找到座位代表房間滿人
		log.Errorf("%s 房間已滿", logger.LOG_Room)
		return false
	}
	// 設定玩家
	r.MutexLock.Lock()
	joinErr := r.DBMatchgame.JoinPlayer(player.DBPlayer.ID)
	if joinErr != nil {
		log.Errorf("%s JoinPlayer時r.DBMatchgame.JoinPlayer(player.DBPlayer.ID)錯誤: %v", logger.LOG_Room, joinErr)
		return false
	}
	log.Infof("安排房間座位: %v", index)
	player.Index = index
	r.Players[index] = player
	r.MutexLock.Unlock()

	r.UpdateMatchgameToDB() // 更新DB
	r.OnRoomPlayerChange()

	log.Infof("%s 玩家%s 已加入房間(%v/%v) 房間資訊: %+v", logger.LOG_Room, player.DBPlayer.ID, r.PlayerCount(), setting.PLAYER_NUMBER, r)
	return true
}

// 將玩家踢出房間
func (r *Room) KickPlayer(conn net.Conn, reason string) {

	seatIndex := r.GetPlayerIndexByTCPConn(conn) // 取得座位索引
	if seatIndex < 0 || r.Players[seatIndex] == nil {
		log.Infof("%s 執行KickPlayer 原因: %s , 玩家已經不在清單中直接返回", logger.LOG_Room, reason)
		return
	}

	log.Infof("%s 執行KickPlayer 原因: %s", logger.LOG_Room, reason)
	player := r.Players[seatIndex]

	// 更新玩家DB
	if player.DBPlayer != nil {
		log.Infof("%s 嘗試踢出玩家 %s", logger.LOG_Room, player.DBPlayer.ID)
		// 取mongoDB player doc
		var mongoPlayerDoc mongo.DBPlayer
		getPlayerDocErr := mongo.GetDocByID(mongo.ColName.Player, player.DBPlayer.ID, &mongoPlayerDoc)
		if getPlayerDocErr != nil {
			log.Errorf("%s 取mongoDB player doc資料發生錯誤: %v", logger.LOG_Room, getPlayerDocErr)
			return
		}
		if !mongoPlayerDoc.RedisSync { // RedisSync為false才需要進行資料同步 如果為true就不用(代表玩家在其他地方已經呼叫了Lobby Server的syncredischeck)
			mongoPlayerDoc.RedisSync = true // 將RedisSync為設為true
			// 更新玩家DB資料
			updatePlayerBson := bson.D{
				{Key: "point", Value: player.DBPlayer.Point},                       // 玩家點數
				{Key: "pointBuffer", Value: player.DBPlayer.PointBuffer},           // 玩家點數溢位
				{Key: "totalWin", Value: player.DBPlayer.TotalWin},                 // 玩家總贏點數
				{Key: "totalExpenditure", Value: player.DBPlayer.TotalExpenditure}, // 玩家總花費點數
				{Key: "leftGameAt", Value: time.Now()},                             // 離開遊戲時間
				{Key: "inMatchgameID", Value: ""},                                  // 玩家不在遊戲房內了
				{Key: "heroExp", Value: player.DBPlayer.HeroExp},                   // 英雄經驗
				{Key: "spellCharges", Value: player.DBPlayer.SpellCharges},         // 技能充能
				{Key: "drops", Value: player.DBPlayer.Drops},                       // 掉落道具
				{Key: "redisSync", Value: player.DBPlayer.RedisSync},               // 設定redisSync為true, 代表已經把這次遊玩結果更新上monogoDB了
			}
			_, updateErr := mongo.UpdateDocByBsonD(mongo.ColName.Player, player.DBPlayer.ID, updatePlayerBson) // 更新DB DBPlayer
			if updateErr != nil {
				log.Errorf("%s 更新玩家 %s DB資料錯誤: %v", logger.LOG_Room, player.DBPlayer.ID, updateErr)
			} else {
				log.Infof("%s 更新玩家 %s DB資料", logger.LOG_Room, player.DBPlayer.ID)
			}
		} else {
			log.Infof("%s 玩家 %s RedisSync為true不需要更新PlayerDoc", logger.LOG_Room, player.DBPlayer.ID)
		}
		r.PubPlayerLeftMsg(player.DBPlayer.ID) // 送玩家離開訊息給Matchmaker
	}
	player.RedisPlayer.ClosePlayer() // 關閉該玩家的RedisDB
	r.MutexLock.Lock()
	r.Players[seatIndex] = nil
	r.DBMatchgame.KickPlayer(player.DBPlayer.ID)
	r.UpdateMatchgameToDB() // 更新房間DB
	r.MutexLock.Unlock()
	player.CloseConnection() // 關閉連線
	r.OnRoomPlayerChange()
	// 廣播玩家離開封包
	r.BroadCastPacket(seatIndex, &packet.Pack{
		CMD: packet.LEAVE_TOCLIENT,
		Content: &packet.Leave_ToClient{
			PlayerIdx: seatIndex,
		},
	})
	// 廣播玩家封包
	r.BroadCastPacket(seatIndex, &packet.Pack{
		CMD:    packet.UPDATEPLAYER_TOCLIENT,
		PackID: -1,
		Content: &packet.UpdatePlayer_ToClient{
			Players: r.GetPacketPlayers(),
		},
	})

	log.Infof("%s 踢出玩家完成", logger.LOG_Room)
}

// 房間人數有異動處理
func (r *Room) OnRoomPlayerChange() {
	if r == nil {
		return
	}
	playerCount := r.PlayerCount()
	// log.Infof("%s 根據玩家數量決定是否升怪 玩家數量: %v", logger.LOG_MonsterSpawner, playerCount)

	if playerCount >= setting.PLAYER_NUMBER { // 滿房
		r.MSpawner.SpawnSwitch(true) // 生怪
	} else if playerCount == 0 { // 空房間處理
		r.MSpawner.SpawnSwitch(false) // 停止生怪
	} else { // 有人但沒有滿房
		r.MSpawner.SpawnSwitch(true) // 生怪
	}
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
	// log.Errorf("//////////////////////////來自player%v(%s) 的 %v 封包", player.Index, player.DBPlayer.ID, pack.CMD)
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
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		r.SetHero(player, content.HeroID, content.HeroSkinID) // 設定使用的英雄ID
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
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		r.KickPlayer(conn, "玩家主動離開") // 將玩家踢出房間

	// ==========發動攻擊==========
	case packet.ATTACK:
		content := packet.Attack{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleAttack(player, pack.PackID, content)
	// ==========擊中怪物==========
	case packet.HIT:
		content := packet.Hit{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleHit(player, pack, content)
	// ==========使用道具==========
	case packet.DROPSPELL:
		content := packet.DropSpell{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleDropSpell(player, pack, content)
	// ==========設定自動攻擊==========
	case packet.AUTO:
		content := packet.Auto{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleAuto(player, pack, content)
	case packet.LVUPSPELL:
		content := packet.LvUpSpell{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s parse %s failed", logger.LOG_Room, pack.CMD)
			return fmt.Errorf("parse %s failed", pack.CMD)
		}
		MyRoom.HandleLvUpSpell(player, pack, content)
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
	// if pack.CMD != packet.SPAWN_TOCLIENT {
	// 	log.Infof("廣播封包給其他玩家 CMD: %v", pack.CMD)
	// }
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
			log.Errorf("%s 廣播封包(%s)錯誤: %v", logger.LOG_Room, pack.CMD, err)
			r.KickPlayer(v.ConnTCP.Conn, "BroadCastPacket錯誤")
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
		r.KickPlayer(r.Players[pIndex].ConnTCP.Conn, "SendPacketToPlayer錯誤")
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
		log.Errorf("%s (UDP)送封包錯誤 %s", logger.LOG_Room, sendErr.Error())
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
			log.Errorf("%s (UDP)送封包錯誤 %s", logger.LOG_Room, sendErr.Error())
			return
		}
	}
}

// 遊戲計時器
func (r *Room) RoomTimer(stop chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s RoomTimer錯誤: %v.\n%s", logger.LOG_Room, err, string(debug.Stack()))
			stop <- struct{}{}
		}
	}()
	ticker := time.NewTicker(time.Duration(TIMELOOP_MILISECS) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			r.GameTime += float64(TIMELOOP_MILISECS) / float64(1000) // 更新遊戲時間
			for _, player := range r.Players {
				if player == nil {
					continue
				}
				nowTime := time.Now()
				// 玩家無心跳超過X秒就踢出遊戲房
				// log.Infof("%s 目前玩家 %s 已經無回應 %.0f 秒了", logger.LOG_Room, player.DBPlayer.ID, nowTime.Sub(player.LastUpdateAt).Seconds())
				if nowTime.Sub(player.LastUpdateAt) > time.Duration(KICK_PLAYER_SECS)*time.Second {
					MyRoom.KickPlayer(player.ConnTCP.Conn, "玩家心跳逾時")
				}
			}
		case <-stop:
			return
		}
	}
}

// 將房間資料更新上DB
func (room *Room) UpdateMatchgameToDB() {
	log.Infof("%s 開始更新Matchgame到DB: %v", logger.LOG_Room, room.DBMatchgame)

	_, err := mongo.UpdateDocByInterface(mongo.ColName.Matchgame, room.DBMatchgame.ID, room.DBMatchgame)
	if err != nil {
		log.Errorf("%s UpdateMatchgameToDB時mongo.UpdateDocByID(mongo.ColName.Matchgame, room.DBMatchgame.ID, updateData)發生錯誤 %v", logger.LOG_Room, err)
	}

	log.Infof("%s 更新Matchgame到DB完成", logger.LOG_Room)
}
