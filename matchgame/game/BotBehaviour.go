package game

import (
	"fmt"
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// 移除所有Bot
func RemoveAllBots(reason string) {
	for _, gamer := range MyRoom.Gamers {
		if bot, ok := gamer.(*Bot); ok {
			MyRoom.KickBot(bot, reason)
		}
	}
	MyRoom.UpdateMatchgameToDB() // 更新房間DB
}

// 加入Bot
func AddBot() *Bot {
	botID := IDAccumulator.GetNextIdx("BotID") // 取下一個BotID

	// 取得隨機英雄
	rndHeroJson, err := gameJson.GetRndHero()
	if err != nil {
		log.Errorf("%s gameJson.GetRndHero()錯誤: %v", logger.LOG_BotBehaviour, err)
		return nil
	}
	heroID, err := strconv.ParseInt(rndHeroJson.ID, 10, 64)
	if err != nil {
		log.Errorf("%s strconv.ParseInt(rndHeroJson.ID,64,64)錯誤: %v", logger.LOG_BotBehaviour, err)
		return nil
	}

	heroSkinID, err := gameJson.GetRndHeroSkinByHeroID(int(heroID)) // 取得隨機英雄Skin
	if err != nil {
		log.Errorf("%s  gameJson.GetRndHeroSkinByHeroID(int(heroID))錯誤: %v", logger.LOG_BotBehaviour, err)
		return nil
	}
	spellJsons := rndHeroJson.GetSpellJsons() // 取得英雄技能
	hero := Hero{
		ID:     int(heroID),
		skinID: heroSkinID,
		spells: spellJsons,
	}

	bot := Bot{
		ID:           botID,
		MyHero:       &hero,
		curTargetIdx: -1, // 無攻擊目標時, curTargetIdx為-1
	}
	bot.InitHero([3]int{}, [3]int{})
	joined := MyRoom.JoinPlayer(&bot)
	if !joined {
		log.Errorf("%s 玩家加入房間失敗", logger.LOG_Main)
		return nil
	}

	// 廣播更新玩家
	MyRoom.BroadCastPacket(-1, &packet.Pack{
		CMD:    packet.UPDATEPLAYER_TOCLIENT,
		PackID: -1,
		Content: &packet.UpdatePlayer_ToClient{
			Players: MyRoom.GetPacketPlayers(),
		},
	})
	// 廣播英雄選擇
	heroIDs, heroSkinIDs := MyRoom.GetHeroInfos()
	MyRoom.BroadCastPacket(-1, &packet.Pack{
		CMD: packet.SETHERO_TOCLIENT,
		Content: &packet.SetHero_ToClient{
			HeroIDs:     heroIDs,
			HeroSkinIDs: heroSkinIDs,
		},
	})
	bot.newSelectTargetLoop()
	return &bot
}
func (bot *Bot) stopAllGoroutine() {
	if bot == nil {
		return
	}
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan()
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan()
	}
}

func (bot *Bot) newSelectTargetLoop() {
	if bot == nil {
		return
	}
	bot.stopAllGoroutine() // 關閉goroutine
	bot.curTargetIdx = -1  // 移除目前鎖定的怪物
	selectLoopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	bot.SelectTargetLoopChan = selectLoopChan
	go bot.selectTargetLoop()
}

func (bot *Bot) selectTargetLoop() {
	if bot == nil {
		return
	}
	ticker := time.NewTicker(gSetting.BOT_TARGET_MS * time.Millisecond)
	defer ticker.Stop()

	targetMonster := bot.selectRndMonsterAsTarget()
	if targetMonster != nil {
		bot.newAtkLoop()
	}

	for {
		select {
		case <-ticker.C:
			if bot.curTargetIdx != -1 {
				continue
			}
			targetMonster := bot.selectRndMonsterAsTarget()
			if targetMonster != nil {
				bot.newAtkLoop()
			}
		case <-bot.SelectTargetLoopChan.StopChan:
			return // 終止goroutine
		}
	}

}

// 設定Bot的目標為隨機怪物
func (bot *Bot) selectRndMonsterAsTarget() *Monster {
	if bot == nil {
		return nil
	}

	rndMonster := utility.GetRndValueFromMap(MyRoom.MSpawner.GetAvailableMonsters())
	if rndMonster != nil {
		bot.curTargetIdx = rndMonster.MonsterIdx
	}
	return rndMonster
}

// 設定Bot的目標為賠率最高的怪物
// func (bot *Bot) selectMaxOddsMonsterAsTarget() *Monster {
// 	if bot == nil {
// 		return nil
// 	}
// 	maxOdds := 0
// 	var curMaxOddsMonster *Monster
// 	availableMonsters := MyRoom.MSpawner.GetAvailableMonsters()
// 	for _, m := range availableMonsters {
// 		if m == nil {
// 			continue
// 		}
// 		if m.Odds > maxOdds {
// 			maxOdds = m.Odds
// 			curMaxOddsMonster = m
// 		}
// 	}
// 	if curMaxOddsMonster != nil {
// 		bot.curTargetIdx = curMaxOddsMonster.MonsterIdx
// 	}
// 	return curMaxOddsMonster
// }

func (bot *Bot) newAtkLoop() {
	if bot == nil {
		return
	}
	bot.stopAllGoroutine() // 關閉goroutine
	atkLoopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	bot.AttackLoopChan = atkLoopChan
	go bot.attackTargetLoop()
}

func (bot *Bot) attackTargetLoop() {
	if bot == nil {
		return
	}
	ticker := time.NewTicker(gSetting.BOT_ATTACK_MS * time.Millisecond)
	changeTargetMiliSec, err := utility.GetRndIntFromRangeStr(gSetting.BOT_CHANGE_TARGET_MS, "~")
	if err != nil {
		log.Errorf("%s utility.GetRndIntFromRangeStr發生錯誤: %v", logger.LOG_BotBehaviour, err)
		return
	}
	attackPassTime := 0
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if bot.curTargetIdx != -1 {
				attackPassTime += gSetting.BOT_ATTACK_MS
				if attackPassTime >= changeTargetMiliSec {
					bot.newSelectTargetLoop()
					return
				} else {
					bot.attack()
				}
			} else {
				continue
			}
		case <-bot.AttackLoopChan.StopChan:
			return
		}
	}
}

func (bot *Bot) attack() {
	if bot == nil {
		return
	}
	curMonster := MyRoom.MSpawner.GetMonster(bot.curTargetIdx)
	if curMonster == nil || curMonster.IsOutOfBoundary() || curMonster.IsLeft() {
		bot.newSelectTargetLoop() // 目標已死亡就重新跑選目標goroutine
		return
	}
	spellJsonID := fmt.Sprintf("%v_attack", bot.MyHero.ID)
	spellJson, err := gameJson.GetHeroSpellByID(spellJsonID)
	if err != nil {
		log.Errorf("%s  gameJson.GetHeroSpellByID(spellJsonID)錯誤: %v", logger.LOG_BotBehaviour, err)
		return
	}
	mIdx := curMonster.MonsterIdx
	dir := curMonster.GetCurVec3Pos().Sub(GetPlayerVec3Pos(bot.Index))
	normalizedDir := dir.Normalize()
	// 送攻擊封包
	MyRoom.BroadCastPacket(-1, &packet.Pack{
		CMD:    packet.ATTACK_TOCLIENT,
		PackID: -1,
		Content: &packet.Attack_ToClient{
			PlayerIdx:   bot.Index,
			SpellJsonID: spellJsonID,
			MonsterIdx:  mIdx,
			AttackLock:  true,
			AttackPos:   []float64{},
			AttackDir:   []float64{normalizedDir.X, normalizedDir.Y, normalizedDir.Z},
		}},
	)

	go bot.RunHit(spellJson)
}

func (bot *Bot) RunHit(spellJson gameJson.HeroSpellJsonData) {
	if bot == nil {
		return
	}
	curMonster := MyRoom.MSpawner.GetMonster(bot.curTargetIdx)
	if curMonster == nil || curMonster.IsOutOfBoundary() || curMonster.IsLeft() {
		return
	}
	hitTime := float64(0)
	spellSpd := spellJson.GetSpellSpeed()
	if spellSpd != 0 {
		dist := bot.calcDistanceFromTarget()
		if dist < 0 {
			return
		}
		hitTime = dist / spellSpd
	}
	if hitTime > 0 {
		time.Sleep(time.Duration(hitTime * float64(time.Second)))
		// 等待後要再確認目標還存在
		curMonster = MyRoom.MSpawner.GetMonster(bot.curTargetIdx)
		if curMonster == nil || curMonster.IsOutOfBoundary() || curMonster.IsLeft() {
			return
		}
	}

	// 計算擊殺
	rtp := float64(0)
	spellType := spellJson.GetSpellType()
	if spellType == "HeroSpell" {
		idx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
		idx -= 1                                           // 表格取出來的技能索引要-1
		if err != nil {
			log.Errorf("%s HandleHit時utility.ExtractLastDigit(spellJson.ID錯誤: %v", logger.LOG_Action, err)
		} else {
			rtp = spellJson.GetRTP(bot.MyHero.GetSpellLV(idx))
		}
	} else if spellType == "DropSpell" {
		rtp = spellJson.GetRTP(1) // 掉落技能只有固定等級1
	}
	// 取波次命中數
	spellMaxHits := spellJson.MaxHits

	killMonsterIdxs := make([]int, 0) // 擊殺怪物索引清單, [1,1,3]就是依次擊殺索引為1,1與3的怪物
	gainPoints := make([]int, 0)      // 獲得點數清單, [1,1,3]就是依次獲得點數1,1與3

	gainSpellCharges := make([]int, 0) // 獲得技能充能清單, [1,1,3]就是依次獲得技能1,技能1,技能3的充能
	// Bot不用獲得英雄經驗
	// gainHeroExps := make([]int32, 0)     // 獲得英雄經驗清單, [1,1,3]就是依次獲得英雄經驗1,1與3
	rndUnchargedSpell, gotUnchargedSpell := bot.GetRandomChargeableSpell() // 計算是否有尚未充滿能的技能, 有的話隨機取一個未充滿能的技能
	gainDrops := make([]int, 0)                                            // 獲得掉落清單, [1,1,3]就是依次獲得DropJson中ID為1,1與3的掉落
	ptBuffer := 0                                                          // 點數溢位

	// 取得怪物掉落道具
	dropAddOdds := 0.0 // 掉落道具增加的總RTP
	// 怪物必須有掉落物才需要考慮怪物掉落
	if curMonster.DropID != 0 {
		// 玩家目前還沒擁有該掉落ID 才需要考慮怪物掉落
		if !bot.IsOwnedDrop(curMonster.DropID) {
			dropAddOdds += float64(curMonster.DropRTP)
		}
	}

	// 計算實際怪物死掉獲得點數
	rewardPoint := int((float64(curMonster.Odds) + dropAddOdds) * float64(MyRoom.DBmap.Bet))

	attackKP := float64(0)
	tmpPTBufferAdd := 0

	if spellType == "Attack" { // 普攻
		// 擊殺判定
		hitData := HitData{
			AttackRTP:  MyRoom.MathModel.GameRTP,
			TargetOdds: float64(curMonster.Odds),
			MaxHit:     int(spellMaxHits),
			// ChargeDrop: gotUnchargedSpell,
			MapBet: MyRoom.DBmap.Bet,
		}
		attackKP, tmpPTBufferAdd = MyRoom.MathModel.GetAttackKP(hitData, bot)
		// log.Infof("======spellMaxHits:%v odds:%v attackKP:%v kill:%v ", spellMaxHits, odds, attackKP, kill)
	} else { // 技能攻擊
		hitData := HitData{
			AttackRTP:  rtp,
			TargetOdds: float64(curMonster.Odds),
			MaxHit:     int(spellMaxHits),
			ChargeDrop: false,
			MapBet:     MyRoom.DBmap.Bet,
		}
		attackKP, tmpPTBufferAdd = MyRoom.MathModel.GetSpellKP(hitData, bot)
		log.Errorf("======spellMaxHits:%v rtp: %v odds:%v attackKP:%v", spellMaxHits, rtp, curMonster.Odds, attackKP)
	}

	kill := utility.GetProbResult(attackKP) // 計算是否造成擊殺
	ptBuffer += tmpPTBufferAdd
	// 如果有擊殺就加到清單中
	if kill {
		gainDrops = append(gainDrops, -1)
		// Bot不用獲得充能
		// 技能充能掉落
		dropChargeP := 0.0
		gainSpellCharges = append(gainSpellCharges, -1)

		if gotUnchargedSpell {
			rndUnchargedSpellRTP := float64(0)

			dropSpellIdx, err := utility.ExtractLastDigit(rndUnchargedSpell.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
			if err != nil {
				log.Errorf("%s HandleHit時utility.ExtractLastDigit(rndUnchargedSpell.ID錯誤: %v", logger.LOG_Action, err)
			} else {
				// log.Errorf("技能ID: %v 索引: %v 技能等級: %v", rndUnchargedSpell.ID, spellIdx, player.MyHero.SpellLVs[spellIdx])
				rndUnchargedSpellRTP = rndUnchargedSpell.GetRTP(bot.GetHero().GetSpellLV(dropSpellIdx - 1))
			}
			// log.Errorf("rndUnchargedSpellRTP: %v", rndUnchargedSpellRTP)
			dropChargeP = MyRoom.MathModel.GetHeroSpellDropP_AttackKilling(rndUnchargedSpellRTP, float64(curMonster.Odds))
			if utility.GetProbResult(dropChargeP) {
				gainSpellCharges[len(gainSpellCharges)-1] = dropSpellIdx
			}
		}
		killMonsterIdxs = append(killMonsterIdxs, curMonster.MonsterIdx)
		gainPoints = append(gainPoints, rewardPoint)
		// Bot不用獲得英雄經驗
		// gainHeroExps = append(gainHeroExps, int32(curMonster.EXP))
		if curMonster.DropID != 0 {
			gainDrops[len(gainDrops)-1] = curMonster.DropID
		}
	}

	// 送擊中封包
	hitPack := packet.Pack{
		CMD:    packet.HIT_TOCLIENT,
		PackID: -1,
		Content: &packet.Hit_ToClient{
			PlayerIdx:        bot.Index,
			KillMonsterIdxs:  killMonsterIdxs,
			GainPoints:       gainPoints,
			GainHeroExps:     []int{},
			GainSpellCharges: gainSpellCharges,
			GainDrops:        gainDrops,
			PTBuffer:         0,
		}}
	MyRoom.SettleHit(bot, hitPack)
}

func (bot *Bot) calcDistanceFromTarget() float64 {
	if bot == nil {
		return -1
	}
	if bot.curTargetIdx == -1 {
		return -1
	}
	curMonster := MyRoom.MSpawner.GetMonster(bot.curTargetIdx)
	if curMonster == nil || curMonster.IsOutOfBoundary() || curMonster.IsLeft() {
		return -1
	}
	return utility.GetDistance(GetPlayerVec2Pos(bot.Index), curMonster.GetCurVec2Pos())
}
