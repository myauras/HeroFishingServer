package game

import (
	"herofishingGoModule/utility"
	"matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type Hero struct {
	HeroID     int           // 英雄ID
	HeroSkinID string        // SkinID
	HeroEXP    int           // 英雄經驗
	Spells     [3]*HeroSpell // 英雄技能
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
