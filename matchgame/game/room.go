package game

import (
	"errors"
	"fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/setting"
	"herofishingGoModule/utility"
	"matchgame/gamemath"
	logger "matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"
	"net"
	"os"
	"runtime/debug"
	"runtime/pprof"
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
	KICK_PLAYER_SECS     float64 = 5   // 最長允許玩家無心跳X秒後踢出遊戲房
	ATTACK_EXPIRED_SECS  float64 = 30  // 攻擊事件實例被創建後X秒後過期(過期代表再次收到同樣的AttackID時Server不會處理)
	UPDATETIMER_MILISECS int     = 500 // 計時器X毫秒跑一次
)

type Room struct {
	// 玩家陣列(索引0~3 分別代表4個玩家)
	// 1. 索引就是玩家的座位, 一進房間後就不會更動 所以HeroIDs[0]就是在座位0玩家的英雄ID
	// 2. 座位無關玩家進來順序 有人離開就會空著 例如 索引2的玩家離開 Players[2]就會是nil 直到有新玩家加入
	Players             [setting.PLAYER_NUMBER]*gSetting.Player // 玩家陣列
	RoomName            string                                  // 房間名稱(也是DB文件ID)(房主UID+時間轉 MD5)
	GameState           GameState                               // 遊戲狀態
	DBMatchgame         *mongo.DBMatchgame                      // DB遊戲房資料
	DBmap               *mongo.DBMap                            // DB地圖設定
	GameTime            float64                                 // 遊戲開始X秒
	ErrorLogs           []string                                // ErrorLogs
	MathModel           *gamemath.Model                         // 數學模型
	MSpawner            *MonsterSpawner                         // 生怪器
	AttackEvents        map[string]*gSetting.AttackEvent        // 攻擊事件
	lastChangeStateTime time.Time                               // 上次更新房間狀態時間
	MutexLock           sync.Mutex
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
	MyRoom.AttackEvents = make(map[string]*gSetting.AttackEvent)

	// 這裡之後要加房間初始化Log到DB

	log.Infof("%s InitGameRoom完成", logger.LOG_Room)
	roomChan <- MyRoom
}
func (r *Room) WriteGameErrorLog(log string) {
	r.ErrorLogs = append(r.ErrorLogs, log)
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
	heroSpells := [3]*gSetting.HeroSpell{}
	for i := 0; i < 3; i++ {
		heroSpells[i] = &gSetting.HeroSpell{
			Charge:    0,
			SpellJson: spellJsons[i],
		}
	}
	if player.MyHero != nil {
		heroEXP = player.MyHero.HeroEXP
	}
	player.MyHero = &gSetting.Hero{
		HeroID:     heroID,
		HeroSkinID: heroSkinID,
		HeroEXP:    heroEXP,
		Spells:     heroSpells,
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
		heroIDs[i] = player.MyHero.HeroID
		heroSkinIDs[i] = player.MyHero.HeroSkinID
	}
	return heroIDs, heroSkinIDs
}

// 把玩家加到房間中, 成功時回傳true
func (r *Room) JoinPlayer(player *gSetting.Player) bool {
	log.Info("JoinPlayer")
	r.MutexLock.Lock()
	defer r.MutexLock.Unlock()
	log.Infof("r.Players: %v", r.Players)
	index := -1
	for i, v := range r.Players {
		if v == nil && index == -1 { // 有座位是空的就把座位索引存起來
			index = i
			break
		}
		if v.DBPlayer.ID == player.DBPlayer.ID { // 如果要加入的玩家ID與目前房間的玩家ID一樣就回傳失敗
			log.Errorf("%s PlayerJoin failed, room exist the same playerID: %v.\n", logger.LOG_Room, player.DBPlayer.ID)
			return false
		}
	}

	if index == -1 { // 沒有找到座位代表房間滿人
		log.Errorf("%s PlayerJoin failed, room is full", logger.LOG_Room)
		return false
	}
	// 設定玩家
	player.Index = index
	r.Players[index] = player
	log.Info("JoinPlayer Finished")
	return true
}

// 將玩家踢出房間
func (r *Room) KickPlayer(lockRoom bool, conn net.Conn) {
	log.Infof("%s 執行KickPlayer", logger.LOG_Room)
	if lockRoom {
		r.MutexLock.Lock()
		defer r.MutexLock.Unlock()
	}
	seatIndex := r.GetPlayerIndexByTCPConn(conn) // 取得座位索引
	if seatIndex < 0 || r.Players[seatIndex] == nil {
		return
	}
	player := r.Players[seatIndex]
	// 更新玩家DB
	if r.Players[seatIndex].DBPlayer != nil {
		log.Infof("%s 踢出玩家 %s", logger.LOG_Room, player.DBPlayer.ID)
		// 更新玩家DB資料
		updatePlayerBson := bson.D{
			{Key: "point", Value: player.DBPlayer.Point},     // 設定玩家點數
			{Key: "heroExp", Value: player.DBPlayer.HeroExp}, // 設定英雄經驗
			{Key: "leftGameAt", Value: time.Now()},           // 設定離開遊戲時間
			{Key: "inMatchgameID", Value: ""},                // 設定玩家不在遊戲房內了
		}
		player.RedisPlayer.ClosePlayer() // 關閉該玩家的RedisDB
		mongo.UpdateDocByID(mongo.ColName.Player, player.DBPlayer.ID, updatePlayerBson)
		log.Infof("%s 更新玩家 %s DB資料玩家", logger.LOG_Room, player.DBPlayer.ID)
	}
	player.CloseConnection()
	r.Players[seatIndex] = nil
	r.UpdatePlayer()
	log.Infof("%s 踢出玩家完成", logger.LOG_Room)
}

func (r *Room) HandleMessage(conn net.Conn, pack packet.Pack, stop chan struct{}) error {
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
			log.Errorf("%s Parse %s Failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("Parse %s Failed", pack.CMD)
		}
		r.SetHero(conn, content.HeroID, content.HeroSkinID) // 設定使用的英雄ID
		heroIDs, heroSkinIDs := r.GetHeroInfos()
		// 廣播給所有玩家
		r.BroadCastPacket(&packet.Pack{ // 廣播封包
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
			log.Errorf("%s Parse %s Failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("Parse %s Failed", pack.CMD)
		}
		r.KickPlayer(true, conn) // 將玩家踢出房間

	// ==========擊中怪物==========
	case packet.HIT:
		content := packet.Hit{}
		if ok := content.Parse(pack.Content); !ok {
			log.Errorf("%s Parse %s Failed", logger.LOG_Main, pack.CMD)
			return fmt.Errorf("Parse %s Failed", pack.CMD)
		}
		MyRoom.HandleAttackEvent(conn, pack, content)
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
func (r *Room) GetPlayerByTCPConn(conn net.Conn) *gSetting.Player {
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
func (r *Room) GetPlayerByConnToken(connToken string) *gSetting.Player {
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
		if (elapsed.Minutes()) >= 3 && r.GameState != End {
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
		switch r.GameState {
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
	r.GameState = state
}

// 送封包給遊戲房間內所有玩家
func (r *Room) BroadCastPacket(pack *packet.Pack) {
	log.Infof("廣播封包 CMD: %v", pack.CMD)
	// 送封包給所有房間中的玩家
	for _, v := range r.Players {
		if v == nil || v.ConnTCP.Conn == nil {
			continue
		}
		err := packet.SendPack(v.ConnTCP.Encoder, pack)
		if err != nil {
			log.Errorf("%s 廣播封包錯誤: %v", logger.LOG_Room, err)
		}
	}
}

// 送封包給玩家
func (r *Room) SendPacketToPlayer(pIndex int, pack *packet.Pack) {
	if r.Players[pIndex] == nil || r.Players[pIndex].ConnTCP.Conn == nil {
		return
	}
	err := packet.SendPack(r.Players[pIndex].ConnTCP.Encoder, pack)
	if err != nil {
		log.Errorf("%s SendPacketToPlayer error: %v", logger.LOG_Room, err)
		r.KickPlayer(true, r.Players[pIndex].ConnTCP.Conn)
	}
}

// 送遊戲房中所有玩家狀態封包
func (r *Room) UpdatePlayer() {
	r.BroadCastPacket(&packet.Pack{
		CMD:    packet.UPDATEPLAYER_TOCLIENT,
		PackID: -1,
		Content: &packet.UpdatePlayer_ToClient{
			Players: r.Players,
		},
	})
}

// 遊戲計時器
func (r *Room) UpdateTimer(stop chan struct{}) {
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
			r.MutexLock.Lock()
			r.GameTime += UPDATE_INTERVAL_MS / 1000 // 更新遊戲時間
			log.Info("UpdateTimer")
			for _, player := range r.Players {
				if player == nil {
					continue
				}
				nowTime := time.Now()
				// 玩家無心跳超過5秒就踢出遊戲房
				log.Infof("%s 目前玩家 %s 已經無回應 %.0f 秒了", logger.LOG_Room, player.DBPlayer.ID, nowTime.Sub(player.LastUpdateAt).Seconds())
				if nowTime.Sub(player.LastUpdateAt) > time.Duration(KICK_PLAYER_SECS)*time.Second {
					MyRoom.KickPlayer(false, player.ConnTCP.Conn)
				}
			}
			r.MutexLock.Unlock()
		case <-stop:
			return
		}
	}
}

// 處理收到的攻擊事件
func (room *Room) HandleAttackEvent(conn net.Conn, pack packet.Pack, hitCMD packet.Hit) {
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
	// 取波次命中數
	spellMaxHits := spellJson.MaxHits
	// 花費點數
	spendPoint := int64(0)

	hitMonsterIdxs := make([]int, 0)   // 擊中怪物索引清單
	killMonsterIdxs := make([]int, 0)  // 擊殺怪物索引清單
	gainPoints := make([]int64, 0)     // 獲得點數清單
	gainSpellCharges := make([]int, 0) // 獲得技能充能清單
	gainHeroExps := make([]int, 0)     // 獲得英雄經驗清單

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

			// 計算實際怪物死掉獲得點數數
			rewardPoint := int64(odds * float64(room.DBmap.Bet))

			// 計算是否造成擊殺
			kill := false
			rndUnchargedSpell := player.MyHero.GetRandomUnchargedSpell()
			if rtp == 0 { // 此攻擊為普攻, RTP為0都歸類在普攻
				spendPoint = -int64(room.DBmap.Bet)
				// 擊殺判定
				attackKP := room.MathModel.GetAttackKP(odds, int(spellMaxHits), rndUnchargedSpell != nil)
				kill = utility.GetProbResult(attackKP)
				log.Infof("======attackID: %s, spellMaxHits:%v odds:%v attackKP:%v kill:%v ", hitCMD.AttackID, spellMaxHits, odds, attackKP, kill)
			} else { // 此攻擊為技能攻擊
				attackKP := room.MathModel.GetSpellKP(rtp, odds, int(spellMaxHits))
				kill = utility.GetProbResult(attackKP)
				log.Infof("======attackID: %s, spellMaxHits:%v rtp: %v odds:%v attackKP:%v kill:%v", hitCMD.AttackID, spellMaxHits, rtp, odds, attackKP, kill)
			}

			// 如果有擊殺就加到清單中
			if kill {
				// 技能充能掉落
				dropChargeP := 0.0
				if rndUnchargedSpell != nil {
					dropChargeP = room.MathModel.GetHeroSpellDropP_AttackKilling(rndUnchargedSpell.SpellJson.RTP, odds)
					if utility.GetProbResult(dropChargeP) {
						spellIndex, err := utility.ExtractLastDigit(rndUnchargedSpell.SpellJson.ID)
						if err != nil {
							log.Errorf("%s utility.ExtractLastDigit(rndUnchargedSpell.SpellJson.ID)錯誤: %v", logger.LOG_Room, err)
						}
						gainSpellCharges = append(gainSpellCharges, spellIndex)
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

	// 以client傳來的AttackID來建立攻擊事件, 如果攻擊事件已存在代表是同一個技能但不同波次的攻擊, 此時就追加擊中怪物清單在該攻擊事件
	if _, ok := room.AttackEvents[hitCMD.AttackID]; !ok {
		// 檢查擊中怪物數量是否超過此技能的最大擊中數量
		if len(hitMonsterIdxs) > int(spellMaxHits) {
			log.Errorf("%s 收到的擊中數量超過此技能最大擊中數量: %v", logger.LOG_Room, err)
			return
		}
		idxs := make([][]int, 1)
		idxs[0] = hitMonsterIdxs
		attackEvent := gSetting.AttackEvent{
			AttackID:    hitCMD.AttackID,
			ExpiredTime: room.GameTime + ATTACK_EXPIRED_SECS,
			MonsterIdxs: idxs,
		}
		room.AttackEvents[hitCMD.AttackID] = &attackEvent
	} else {
		attackEvent := room.AttackEvents[hitCMD.AttackID]
		if attackEvent == nil {
			log.Errorf("%s room.AttackEvents[hitCMD.AttackID]為nil", logger.LOG_Room)
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
		}
		attackEvent.MonsterIdxs = append(attackEvent.MonsterIdxs, hitCMD.MonsterIdxs)
	}
	// 玩家點數變化
	totalGainPoint := spendPoint + utility.SliceSum(gainPoints) // 總點數變化是消耗點數+獲得點數
	if totalGainPoint != 0 {
		player.AddPoint(totalGainPoint)
	}
	// 從怪物清單中移除被擊殺的怪物
	utility.RemoveFromMapByKeys(room.MSpawner.Monsters, killMonsterIdxs)
	// 玩家英雄增加經驗
	player.AddHeroExp(utility.SliceSum(gainHeroExps))

	// log.Infof("killMonsterIdxs: %v \n", killMonsterIdxs)
	// log.Infof("gainPoints: %v \n", gainPoints)
	// log.Infof("gainSpellCharges: %v \n", gainSpellCharges)
	// 廣播給client
	room.BroadCastPacket(&packet.Pack{
		CMD:    packet.HIT_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.Hit_ToClient{
			KillMonsterIdxs:  killMonsterIdxs,
			GainPoints:       gainPoints,
			GainHeroExps:     gainHeroExps,
			GainSpellCharges: gainSpellCharges,
			GainDrops:        make([]int, 0),
		}},
	)
}
