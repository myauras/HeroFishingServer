package packet

import (
	logger "matchgame/logger"
	"reflect"

	log "github.com/sirupsen/logrus"
)

// 攻擊
type Attack struct {
	CMDContent
	AttackID    int    // 攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	SpellJsonID string // 技能表ID
	MonsterIdx  int    // 目標怪物索引, -1就是沒有指定目標
	// 攻擊施放需要的參數(位置, 角度等)
	AttackLock bool      // 是否為鎖定攻擊
	AttackPos  []float64 // 攻擊施放位置
	AttackDir  []float64 // 攻擊施放方向
}

// 攻擊回傳client
type Attack_ToClient struct {
	CMDContent
	PlayerIdx   int    // 玩家座位索引
	SpellJsonID string // 技能表ID
	MonsterIdx  int    // 目標怪物索引
	// 攻擊施放需要的參數(位置, 角度等)
	AttackLock bool      // 是否為鎖定攻擊
	AttackPos  []float64 // 攻擊施放位置
	AttackDir  []float64 // 攻擊施放方向
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
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Attack")
		return false
	}
	if monsterIdx, ok := m["MonsterIdx"].(float64); ok {
		p.MonsterIdx = int(monsterIdx)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse monsterIdx資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Attack")
		return false
	}
	if attackLock, ok := m["AttackLock"].(bool); ok {
		p.AttackLock = attackLock
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse AttackLock資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Attack")
		return false
	}

	if attackPos, ok := m["AttackPos"].([]interface{}); ok {
		var pos []float64
		for _, idx := range attackPos {
			if floatIdx, ok := idx.(float64); ok {
				pos = append(pos, floatIdx)
			} else {
				log.WithFields(log.Fields{
					"invalidType":  reflect.TypeOf(idx),
					"invalidValue": idx,
					"log":          "parse AttackPos資料錯誤",
				}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Attack")
				return false
			}
		}
		p.AttackPos = pos
	}

	if attackDir, ok := m["AttackDir"].([]interface{}); ok {
		var pos []float64
		for _, idx := range attackDir {
			if floatIdx, ok := idx.(float64); ok {
				pos = append(pos, floatIdx)
			} else {
				log.WithFields(log.Fields{
					"invalidType":  reflect.TypeOf(idx),
					"invalidValue": idx,
					"log":          "parse AttackDir資料錯誤",
				}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Attack")
				return false
			}
		}
		p.AttackDir = pos
	}

	return true

}
