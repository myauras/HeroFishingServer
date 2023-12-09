package gamemath

import (
	"matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 模型結構
type Model struct {
	GameRTP        float64
	SpellSharedRTP float64
}

// 取得普攻擊殺率
func (model *Model) GetAttackKP(targetOdds float64, spellMaxHit int, chargeShareRTP bool) float64 {
	attackRTP := model.GameRTP
	if chargeShareRTP { // 需要把普通攻擊的部分RTP分給技能充能掉落時
		attackRTP -= model.SpellSharedRTP
	}
	if attackRTP <= 0 {
		log.Errorf("%s GetAttackKP錯誤 attackRTP<=0", logger.LOG_MathModel)
		return 0
	}
	p := attackRTP / targetOdds / float64(spellMaxHit)

	// 擊殺率大於1時處理
	if p > 1 {
		p = 1
		log.Infof("%s GetAttackKP的p>1", logger.LOG_MathModel)
	}
	return p
}

// 取得技能擊殺率
func (model *Model) GetSpellKP(spellRTP float64, targetOdds float64, spellMaxHit int) float64 {
	p := spellRTP / targetOdds / float64(spellMaxHit)

	// 擊殺率大於1時處理
	if p > 1 {
		p = 1
		log.Infof("%s GetAttackKP的p>1", logger.LOG_MathModel)
	}
	return p
}

// 取得普攻擊殺掉落英雄技能機率
func (model *Model) GetHeroSpellDropP_AttackKilling(spellRTP float64, targetOdds float64) float64 {
	if spellRTP <= model.SpellSharedRTP {
		log.Errorf("%s HeroSpellDropP錯誤 spellRTP<=model.AttackRTP", logger.LOG_MathModel)
		return 0
	}
	attackRTP := model.GameRTP - model.SpellSharedRTP
	p := ((model.GameRTP - attackRTP) / (spellRTP - attackRTP)) / (model.GameRTP / targetOdds)

	// 掉落率大於1時處理
	if p > 1 {
		p = 1
		log.Infof("%s GetHeroSpellDropP_AttackKilling的p>1", logger.LOG_MathModel)
	}
	return p
}
