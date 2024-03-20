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

// 加入Bot
func AddBot() *Bot {
	newBotIdx := IDAccumulator.GetNextIdx("BotIdx") // 自動Bot Idx取下一個bot

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
		Index:  newBotIdx,
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

func (bot *Bot) NewSelectTargetLoop() {
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan() // 關閉原本得goroutine
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan() // 開始搜尋新目標就關閉攻擊goroutine
	}
	selectLoopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	bot.SelectTargetLoopChan = selectLoopChan
	go bot.SelectTargetLoop()
}

func (bot *Bot) SelectTargetLoop() {
	log.Errorf("開始找目標Loop")
	ticker := time.NewTicker(gSetting.BOT_TARGET_MS * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-bot.SelectTargetLoopChan.StopChan:
			return // 終止goroutine
		default:
			log.Errorf("Bot找目標")
			if bot.CurTarget != nil {
				continue
			}
			targetMonster := bot.SelectRndMonsterAsTarget()
			if targetMonster != nil {
				log.Errorf("找到目標怪物 %v", targetMonster.MonsterIdx)
				bot.NewAtkLoop()
			}
		}
	}
}

// 設定Bot的目標為隨機怪物
func (bot *Bot) SelectRndMonsterAsTarget() *Monster {
	var curMaxOddsMonster *Monster
	rndMonster := utility.GetRndValueFromMap(MyRoom.MSpawner.Monsters)
	if rndMonster != nil {
		bot.CurTarget = rndMonster
	}
	return curMaxOddsMonster
}

// 設定Bot的目標為賠率最高的怪物
func (bot *Bot) SelectMaxOddsMonsterAsTarget() *Monster {
	maxOdds := int64(0)
	var curMaxOddsMonster *Monster
	for _, m := range MyRoom.MSpawner.Monsters {
		if m == nil {
			continue
		}
		odds, err := strconv.ParseInt(m.MonsterJson.Odds, 10, 64)
		if err != nil {
			continue
		}
		if odds > maxOdds {
			maxOdds = odds
			curMaxOddsMonster = m
		}
	}
	if curMaxOddsMonster != nil {
		bot.CurTarget = curMaxOddsMonster
	}
	return curMaxOddsMonster
}

func (bot *Bot) NewAtkLoop() {
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan() // 開始攻擊goroutine就可以關閉選目標的goroutine
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan() // 關閉原本得goroutine
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
	ticker := time.NewTicker(gSetting.BOT_ATTACK_MS * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-bot.AttackLoopChan.StopChan:
			return // 終止goroutine
		default:
			bot.Attack()
		}
	}
}

func (bot *Bot) Attack() {
	if monster := MyRoom.MSpawner.GetMonster(bot.CurTarget.MonsterIdx); monster == nil {
		bot.CurTarget = nil // 移除目前鎖定的怪物
		log.Errorf("目標怪物已死亡, 重新尋找目標")
		bot.NewSelectTargetLoop() // 目標已死亡就重新跑選目標goroutine
		return
	}
	log.Errorf("對怪物 %v 進行攻擊", bot.CurTarget.MonsterIdx)
	spellJsonID := fmt.Sprintf("%v_attack", bot.MyHero.ID)
	mIdx := bot.CurTarget.MonsterIdx
	log.Errorf("spellJsonID: %s", spellJsonID)
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

func (bot *Bot) NewChangeTargetLoop() {
	if bot.SelectTargetLoopChan != nil {
		bot.SelectTargetLoopChan.ClosePackReadStopChan() // 開始攻擊goroutine就可以關閉選目標的goroutine
	}
	if bot.AttackLoopChan != nil {
		bot.AttackLoopChan.ClosePackReadStopChan() // 關閉原本得goroutine
	}
	atkLoopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	bot.AttackLoopChan = atkLoopChan
	go bot.AttackTargetLoop()
}
