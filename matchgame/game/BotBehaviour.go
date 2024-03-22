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
		SkinID: heroSkinID,
		Spells: spellJsons,
	}

	bot := Bot{
		ID:     botID,
		MyHero: &hero,
	}
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
	bot.NewSelectTargetLoop()
	return &bot
}
func (bot *Bot) StopAllGoroutine() {
	if bot == nil {
		return
	}
	log.Errorf("StopAllGoroutine")
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan()
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan()
	}
}

func (bot *Bot) NewSelectTargetLoop() {
	if bot == nil {
		return
	}
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan()
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan()
	}
	bot.CurTarget = nil // 移除目前鎖定的怪物
	selectLoopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	bot.SelectTargetLoopChan = selectLoopChan
	go bot.SelectTargetLoop()
}

func (bot *Bot) SelectTargetLoop() {
	if bot == nil {
		return
	}
	ticker := time.NewTicker(gSetting.BOT_TARGET_MS * time.Millisecond)
	defer ticker.Stop()

	targetMonster := bot.SelectRndMonsterAsTarget()
	if targetMonster != nil {
		bot.NewAtkLoop()
	}

	for range ticker.C {
		select {
		case <-bot.SelectTargetLoopChan.StopChan:
			return // 終止goroutine
		default:
			if bot.CurTarget != nil {
				continue
			}
			targetMonster := bot.SelectRndMonsterAsTarget()
			if targetMonster != nil {
				bot.NewAtkLoop()
			}
		}
	}
}

// 設定Bot的目標為隨機怪物
func (bot *Bot) SelectRndMonsterAsTarget() *Monster {
	if bot == nil {
		return nil
	}
	rndMonster := utility.GetRndValueFromMap(MyRoom.MSpawner.Monsters)
	if rndMonster != nil {
		bot.CurTarget = rndMonster
	}
	return rndMonster
}

// 設定Bot的目標為賠率最高的怪物
func (bot *Bot) SelectMaxOddsMonsterAsTarget() *Monster {
	if bot == nil {
		return nil
	}
	maxOdds := 0
	var curMaxOddsMonster *Monster
	for _, m := range MyRoom.MSpawner.Monsters {
		if m == nil {
			continue
		}
		if m.Odds > maxOdds {
			maxOdds = m.Odds
			curMaxOddsMonster = m
		}
	}
	if curMaxOddsMonster != nil {
		bot.CurTarget = curMaxOddsMonster
	}
	return curMaxOddsMonster
}

func (bot *Bot) NewAtkLoop() {
	if bot == nil {
		return
	}
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan()
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan()
	}
	atkLoopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	bot.AttackLoopChan = atkLoopChan
	go bot.AttackTargetLoop()
}

func (bot *Bot) AttackTargetLoop() {
	log.Errorf("開始攻擊Loop")
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
	for range ticker.C {
		select {
		case <-bot.AttackLoopChan.StopChan:
			log.Errorf("終止AttackTargetLoop")
			return // 終止goroutine
		default:
			attackPassTime += gSetting.BOT_ATTACK_MS
			if attackPassTime >= changeTargetMiliSec {
				bot.NewSelectTargetLoop()
				return
			} else {
				bot.Attack()
			}
		}
	}
}

func (bot *Bot) Attack() {
	if bot == nil {
		return
	}
	curMonster := MyRoom.MSpawner.GetMonster(bot.CurTarget.MonsterIdx)
	if curMonster == nil || curMonster.IsOutOfBoundary() || curMonster.IsLeft() {
		bot.NewSelectTargetLoop() // 目標已死亡就重新跑選目標goroutine
		return
	}
	spellJsonID := fmt.Sprintf("%v_attack", bot.MyHero.ID)
	spellJson, err := gameJson.GetHeroSpellByID(spellJsonID)
	if err != nil {
		log.Errorf("%s  gameJson.GetHeroSpellByID(spellJsonID)錯誤: %v", logger.LOG_BotBehaviour, err)
		return
	}
	mIdx := bot.CurTarget.MonsterIdx

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
			AttackDir:   []float64{},
		}},
	)

	// 計算擊殺
	rtp := float64(0)
	spellType := spellJson.GetSpellType()
	if spellType == "HeroSpell" {
		idx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
		if err != nil {
			log.Errorf("%s HandleHit時utility.ExtractLastDigit(spellJson.ID錯誤: %v", logger.LOG_Action, err)
		} else {
			rtp = spellJson.GetRTP(bot.MyHero.SpellLVs[idx])
		}
	} else if spellType == "DropSpell" {
		rtp = spellJson.GetRTP(1) // 掉落技能只有固定等級1
	}
	// 取波次命中數
	spellMaxHits := spellJson.MaxHits

	killMonsterIdxs := make([]int, 0)    // 擊殺怪物索引清單, [1,1,3]就是依次擊殺索引為1,1與3的怪物
	gainPoints := make([]int64, 0)       // 獲得點數清單, [1,1,3]就是依次獲得點數1,1與3
	gainSpellCharges := make([]int32, 0) // 獲得技能充能清單, [1,1,3]就是依次獲得技能1,技能1,技能3的充能
	gainHeroExps := make([]int32, 0)     // 獲得英雄經驗清單, [1,1,3]就是依次獲得英雄經驗1,1與3
	gainDrops := make([]int32, 0)        // 獲得掉落清單, [1,1,3]就是依次獲得DropJson中ID為1,1與3的掉落
	ptBuffer := int64(0)                 // 點數溢位

	// 取得怪物掉落道具
	dropAddOdds := 0.0 // 掉落道具增加的總RTP
	// 怪物必須有掉落物才需要考慮怪物掉落
	if curMonster.DropID != 0 {
		// 玩家目前還沒擁有該掉落ID 才需要考慮怪物掉落
		if !bot.IsOwnedDrop(int32(curMonster.DropID)) {
			dropAddOdds += float64(curMonster.DropRTP)
		}
	}

	// 計算實際怪物死掉獲得點數
	rewardPoint := int64((float64(curMonster.Odds) + dropAddOdds) * float64(MyRoom.DBmap.Bet))

	rndUnchargedSpell, gotUnchargedSpell := bot.GetRandomChargeableSpell() // 計算是否有尚未充滿能的技能, 有的話隨機取一個未充滿能的技能

	attackKP := float64(0)
	tmpPTBufferAdd := int64(0)

	if spellType == "Attack" { // 普攻
		// 擊殺判定
		hitData := HitData{
			AttackRTP:  MyRoom.MathModel.GameRTP,
			TargetOdds: float64(curMonster.Odds),
			MaxHit:     int(spellMaxHits),
			ChargeDrop: gotUnchargedSpell,
			MapBet:     MyRoom.DBmap.Bet,
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
		attackKP, tmpPTBufferAdd = MyRoom.MathModel.GetSpellKP(hitData, player)
		log.Errorf("======spellMaxHits:%v rtp: %v odds:%v attackKP:%v", spellMaxHits, rtp, curMonster.Odds, attackKP)
	}

	kill := utility.GetProbResult(attackKP) // 計算是否造成擊殺
	ptBuffer += tmpPTBufferAdd
	// 如果有擊殺就加到清單中
	if kill {
		// 技能充能掉落
		dropChargeP := 0.0
		gainSpellCharges = append(gainSpellCharges, -1)
		gainDrops = append(gainDrops, -1)
		if gotUnchargedSpell {
			rndUnchargedSpellRTP := float64(0)

			dropSpellIdx, err := utility.ExtractLastDigit(rndUnchargedSpell.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
			if err != nil {
				log.Errorf("%s HandleHit時utility.ExtractLastDigit(rndUnchargedSpell.ID錯誤: %v", logger.LOG_Action, err)
			} else {
				// log.Errorf("技能ID: %v 索引: %v 技能等級: %v", rndUnchargedSpell.ID, spellIdx, player.MyHero.SpellLVs[spellIdx])
				rndUnchargedSpellRTP = rndUnchargedSpell.GetRTP(player.MyHero.SpellLVs[dropSpellIdx])
			}
			// log.Errorf("rndUnchargedSpellRTP: %v", rndUnchargedSpellRTP)
			dropChargeP = MyRoom.MathModel.GetHeroSpellDropP_AttackKilling(rndUnchargedSpellRTP, float64(curMonster.Odds))
			if utility.GetProbResult(dropChargeP) {
				gainSpellCharges[len(gainSpellCharges)-1] = int32(dropSpellIdx)
			}
		}
		// log.Errorf("擊殺怪物: %v", monsterIdx)
		killMonsterIdxs = append(killMonsterIdxs, monsterIdx)
		gainPoints = append(gainPoints, rewardPoint)
		gainHeroExps = append(gainHeroExps, int32(curMonster.EXP))
		if curMonster.DropID != 0 {
			gainDrops[len(gainDrops)-1] = int32(curMonster.DropID)
		}
	}

	// 送擊中封包
	MyRoom.BroadCastPacket(-1, &packet.Pack{
		CMD:    packet.HIT_TOCLIENT,
		PackID: -1,
		Content: &packet.Hit_ToClient{
			PlayerIdx:        bot.Index,
			KillMonsterIdxs:  killMonsterIdxs,
			GainPoints:       gainPoints,
			GainHeroExps:     []int32{},
			GainSpellCharges: []int32{},
			GainDrops:        gainDrops,
			PTBuffer:         0,
		}},
	)
}
