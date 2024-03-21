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
	if bot.CurTarget == nil || MyRoom.MSpawner.GetMonster(bot.CurTarget.MonsterIdx) == nil {
		log.Errorf("目標怪物已死亡, 重新尋找目標")
		bot.NewSelectTargetLoop() // 目標已死亡就重新跑選目標goroutine
		return
	}
	spellJsonID := fmt.Sprintf("%v_attack", bot.MyHero.ID)
	mIdx := bot.CurTarget.MonsterIdx
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
}
