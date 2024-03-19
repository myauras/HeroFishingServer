package game

import (
	"fmt"
	"herofishingGoModule/gameJson"
	"matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"
	"strconv"
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
	log.Errorf("%s  廣播更新玩家", logger.LOG_BotBehaviour)
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
	return &bot
}

func (bot *Bot) SelectTargetLoop() {
	ticker := time.NewTicker(gSetting.BOT_TARGET_MS * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if bot.CurTarget != nil {
			continue
		}
		bot.SelectMaxOddsMonsterAsTarget()
	}
}

// 設定Bot的目標為賠率最高的怪物
func (bot *Bot) SelectMaxOddsMonsterAsTarget() {
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
}

func (bot *Bot) AttackTargetLoop() {
	ticker := time.NewTicker(gSetting.BOT_ATTACK_MS * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if bot.CurTarget == nil {
			continue
		}
		bot.Attack()
	}
}

func (bot *Bot) Attack() {
	// 廣播給client
	// 取技能表
	spellJsonID := fmt.Sprintf("%s_attack", bot.MyHero.ID)
	log.Errorf("spellJsonID: %s", spellJsonID)
	MyRoom.BroadCastPacket(-1, &packet.Pack{
		CMD:    packet.ATTACK_TOCLIENT,
		PackID: -1,
		Content: &packet.Attack_ToClient{
			PlayerIdx:   bot.Index,
			SpellJsonID: spellJsonID,
			MonsterIdx:  bot.CurTarget.MonsterIdx,
			AttackLock:  true,
			AttackPos:   []float64{},
			AttackDir:   []float64{},
		}},
	)
}
