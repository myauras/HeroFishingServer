package game

import (
	"herofishingGoModule/utility"
	"matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type Hero struct {
	ID     int           // 英雄ID
	SkinID string        // SkinID
	EXP    int           // 英雄經驗
	Spells [3]*HeroSpell // 英雄技能
}

// 取得此英雄隨機尚未充滿能的技能
func (hero *Hero) GetRandomUnchargedSpell() *HeroSpell {
	spells := hero.GetUnchargedSpells()
	if len(spells) == 0 {
		return nil
	}
	spell, err := utility.GetRandomTFromSlice(spells)
	if err != nil {
		log.Errorf("%s utility.GetRandomTFromSlice(spells)錯誤: %v", logger.LOG_Setting, err)
	}
	return spell
}

// 取得此英雄尚未充滿能的技能
func (hero *Hero) GetUnchargedSpells() []*HeroSpell {
	spells := make([]*HeroSpell, 0)
	for _, spell := range hero.Spells {
		if !spell.IsCharged() {
			spells = append(spells, spell)
		}
	}
	return spells
}

// 英雄施法充能歸0
func (hero *Hero) ResetHeroSpellCharge(idx int) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s uAddHeroSpellCharge傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return
	}
	hero.Spells[(idx - 1)].Charge = 0 // Spells索引是存0~2所以idx要-1
	log.Infof("重置玩家技能-%v的充能", idx)
}

// 英雄施法充能增減, 傳入1~3
func (hero *Hero) AddHeroSpellCharge(idx int, value int) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s AddHeroSpellCharge傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return
	}
	hero.Spells[(idx - 1)].Charge += value // Spells索引是存0~2所以idx要-1
	log.Infof("玩家技能-%v的充能+%v", idx, value)
}

// 檢查是否可以施法
func (hero *Hero) CanSpell(idx int) bool {
	if idx < 1 || idx > 3 {
		log.Errorf("%s CanSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return false
	}
	cost := hero.Spells[(idx - 1)].SpellJson.Cost // Spells索引是存0~2所以idx要-1

	return hero.Spells[(idx-1)].Charge >= cost
}
