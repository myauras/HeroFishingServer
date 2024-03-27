package game

import (
	"fmt"
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"

	log "github.com/sirupsen/logrus"
)

// 電腦玩家
type Bot struct {
	ID                   int                 // Bot編號 等同於Player的PlaeyrID
	Index                int                 // 玩家在房間的索引(座位)
	MyHero               *Hero               // 使用中的英雄
	GainPoint            int                 // 此玩家在遊戲房總共贏得點數
	PlayerBuffs          []packet.PlayerBuff // 玩家Buffers
	Point                int
	SpellCharges         [3]int
	Drops                [3]int
	curTargetIdx         int
	SelectTargetLoopChan *gSetting.LoopChan
	AttackLoopChan       *gSetting.LoopChan
}

// 初始化玩家英雄
func (bot *Bot) InitHero(spellLVs [3]int, spellCharges [3]int) {
	if bot == nil {
		return
	}

	bot.MyHero = &Hero{
		spellLVs:     spellLVs,
		spellCharges: spellCharges,
	}
}

// 取得ID
func (bot *Bot) GetID() string {
	return fmt.Sprintf("Bot%v", bot.ID)
}

// 設定座位
func (bot *Bot) SetIdx(idx int) {
	bot.Index = idx
}

// 取得座位
func (bot *Bot) GetIdx() int {
	return bot.Index
}

// 取得Hero
func (bot *Bot) GetHero() *Hero {
	return bot.MyHero
}

// 取得玩家Buffers
func (bot *Bot) GetBuffers() []packet.PlayerBuff {
	return bot.PlayerBuffs
}

// 設定玩家Buffers
func (bot *Bot) SetBuffers(buffers []packet.PlayerBuff) {
	bot.PlayerBuffs = buffers
}

// 取得點數
func (bot *Bot) GetPoint() int {
	return bot.Point
}

// 玩家點數增減
func (Bot *Bot) AddPoint(value int) {
	Bot.Point += int(value)

	// 設定玩家本場贏得點數
	if value > 0 {
		Bot.GainPoint += value
	}
}

// 取得本場遊戲獲得點數
func (Bot *Bot) GetGainPoint() int {
	return Bot.GainPoint
}

// 英雄經驗增減
func (Bot *Bot) AddHeroExp(value int) {
	// 不需處理
}

// 技能充能增減, idx傳入1~3
func (Bot *Bot) AddSpellCharge(idx int, value int) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s AddSpellCharge傳入錯誤索引: %v", logger.LOG_Player, idx)
		return
	}
	if value == 0 {
		log.Errorf("%s AddSpellCharge傳入值為0", logger.LOG_Player)
		return
	}
	Bot.SpellCharges[(idx - 1)] = value
}

// 新增掉落
func (Bot *Bot) AddDrop(value int) {
	if value == 0 {
		log.Errorf("%s AddDrop傳入值為0", logger.LOG_Player)
		return
	}
	if Bot.IsOwnedDrop(value) {
		log.Errorf("%s AddDrop時已經有此掉落道具, 無法再新增: %v", logger.LOG_Player, value)
		return
	}
	dropIdx := -1
	for i, v := range Bot.Drops {
		if v == 0 {
			dropIdx = i
			break
		}
	}
	if dropIdx == -1 {
		log.Errorf("%s AddDrop時dropIdx為-1", logger.LOG_Player)
		return
	}
	// log.Infof("%s 電腦%s獲得Drop idx:%v  dropID:%v", logger.LOG_Player, Bot.Index, dropIdx, Bot.Drops[dropIdx])
	Bot.Drops[dropIdx] = value
}

// 移除掉落
func (Bot *Bot) RemoveDrop(value int) {
	if value == 0 {
		log.Errorf("%s RemoveDrop傳入值為0", logger.LOG_Player)
		return
	}
	if !Bot.IsOwnedDrop(value) {

		return
	}
	dropIdx := -1
	for i, v := range Bot.Drops {
		if v == value {
			dropIdx = i
			break
		}
	}
	if dropIdx == -1 {
		log.Errorf("%s RemoveDrop時無此掉落道具, 無法移除: %v", logger.LOG_Player, value)
		log.Errorf("%s RemoveDrop時dropIdx為-1", logger.LOG_Player)
		return
	}
	// log.Infof("%s 電腦%s移除Drop idx:%v  dropID:%v", logger.LOG_Player, Bot.Index, dropIdx, Bot.Drops[dropIdx])
	Bot.Drops[dropIdx] = 0
}

// 是否已經擁有此道具
func (Bot *Bot) IsOwnedDrop(value int) bool {
	for _, v := range Bot.Drops {
		if v == value {
			return true
		}
	}
	return false
}

// 取得此英雄隨機尚未充滿能且已經學習過的技能, 無適合的技能時會返回false
func (Bot *Bot) GetRandomChargeableSpell() (gameJson.HeroSpellJsonData, bool) {
	spells := Bot.GetLearnedAndChargeableSpells()

	if len(spells) == 0 {
		return gameJson.HeroSpellJsonData{}, false
	}
	spell, err := utility.GetRandomTFromSlice(spells)
	if err != nil {
		log.Errorf("%s utility.GetRandomTFromSlice(spells)錯誤: %v", logger.LOG_Player, err)
		return gameJson.HeroSpellJsonData{}, false
	}
	return spell, true
}

// 取得此英雄所有尚未充滿能且已經學習過的技能
func (Bot *Bot) GetLearnedAndChargeableSpells() []gameJson.HeroSpellJsonData {
	spells := make([]gameJson.HeroSpellJsonData, 0)
	if Bot == nil {
		return spells
	}
	for i, v := range Bot.SpellCharges {
		if Bot.MyHero.GetSpellLV(i) <= 0 { // 尚未學習的技能就跳過
			continue
		}
		spell, err := Bot.MyHero.GetSpellJson(i)
		if err != nil {
			log.Errorf("%s GetUnchargedSpells時GetUnchargedSpells錯誤: %v", logger.LOG_Player, err)
			continue
		}
		if v < spell.Cost {
			spells = append(spells, spell)
		}
	}
	return spells
}

// 檢查是否可以施法
func (Bot *Bot) CanSpell(idx int) bool {

	spell, err := Bot.MyHero.GetSpellJson(idx)
	if err != nil {
		return false
	}
	cost := spell.Cost

	return Bot.SpellCharges[(idx-1)] >= cost
}

// 取得普攻CD
func (bot *Bot) GetAttackCDBuff() float64 {
	return 0
}

func (bot *Bot) CloseConnection() {
	// 不需處理
}
