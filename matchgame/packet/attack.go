package packet

import (
	log "github.com/sirupsen/logrus"
	logger "matchgame/logger"
)

// 攻擊
type Attack struct {
	CMDContent
	AttackID    int    // 攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	SpellJsonID string // 技能表ID
	MonsterIdx  int    // 目標怪物索引
	// 攻擊施放需要的參數(位置, 角度等)
}

// 攻擊回傳client
type Attack_ToClient struct {
	CMDContent
	PlayerIdx   int    // 玩家座位
	SpellJsonID string // 技能表ID
	MonsterIdx  int    // 目標怪物索引
	// 攻擊施放需要的參數(位置, 角度等)
}

func (p *Attack) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})
	if attackID, ok := m["AttackID"].(float64); ok {
		p.AttackID = int(attackID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse attackID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Attack")
		return false
	}
	if spellJsonID, ok := m["SpellJsonID"].(string); ok {
		p.SpellJsonID = spellJsonID
	} else {
		log.WithFields(log.Fields{
			"log": "parse SpellJsonID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Hit")
		return false
	}

	return true

}
