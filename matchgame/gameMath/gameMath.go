package gamemath

import (
	"matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 模型結構
type Model struct {
	GameRTP   float64
	AttackRTP float64
}

// 取得普攻擊殺率
func (model *Model) GetAttackKP(targetOdds float64) float64 {
	p := model.AttackRTP / targetOdds
	return p
}

// 取得技能擊殺率
func (model *Model) GetSpellKP(spellRTP float64, targetOdds float64) float64 {
	p := spellRTP / targetOdds
	return p
}

// 英雄技能掉落機率
func (model *Model) HeroSpellDropP(spellRTP float64, targetOdds float64) float64 {
	if spellRTP <= model.AttackRTP {
		log.Errorf("%s HeroSpellDropP錯誤 spellRTP<=model.AttackRTP", logger.LOG_MathModel)
		return 0
	}
	p := ((model.GameRTP - model.AttackRTP) / (spellRTP - model.AttackRTP)) / (model.GameRTP / targetOdds)
	return p
}
