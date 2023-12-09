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

// 英雄施法充能增減
func (hero *Hero) AddHeroSpellCharge(idx int, value int) {
	if idx >= len(hero.Spells) {
		return
	}
	hero.Spells[idx].Charge += value
}

// 檢查是否可以施法
func (hero *Hero) CheckCanSpell(idx int) bool {
	if idx >= len(hero.Spells) {
		return false
	}
	if hero.Spells[idx].Charge < hero.Spells[idx].SpellJson.Cost {
		return false
	}
	hero.AddHeroSpellCharge(idx, -hero.Spells[idx].SpellJson.Cost)
	return true
}
