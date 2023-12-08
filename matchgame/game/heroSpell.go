package game

import (
	// log "github.com/sirupsen/logrus"
	"herofishingGoModule/gameJson"

	// "matchgame/logger"
)

// 英雄技能
type HeroSpell struct {
	SpellJson gameJson.HeroSpellJsonData
	Charge    int // 充能
}

// 取得此技能充能比例
func (spell *HeroSpell) GetChargeRatio() float64 {
	return float64(spell.Charge) / float64(spell.SpellJson.Cost)
}

// 此技能是否充滿能
func (spell *HeroSpell) IsCharged() bool {
	return spell.Charge >= spell.SpellJson.Cost
}
