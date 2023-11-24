package packet

import (
	logger "matchgame/logger"
	"reflect"

	log "github.com/sirupsen/logrus"
)

// 命中怪物
type Action_HitCMD struct {
	CMDContent
	// 攻擊ID格式為 [玩家index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	AttackID    string // 攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	MonsterIdxs []int  // 此次命中怪物索引清單
	SpellJsonID string // 技能表ID
}

// 命中怪物回傳client
type Action_HitCMD_Reply struct {
	CMDContent
	// KillMonsterIdxs與GainGolds是對應的, 例如KillMonsterIdxs為[0,3,6]而GainGolds是[30,0,120], 就是此次攻擊擊殺了索引為0,3,6的怪物並分別獲得30,0,120金幣
	KillMonsterIdxs  []int   // 擊殺怪物索引清單
	GainGolds        []int64 // 獲得金幣清單
	GainSpellCharges []int   // 獲得技能充能清單, [1,2]就是第一個技能跟第二個技能
	GainDrops        []int   // 獲得掉落清單, [4,5]就是DropJsonID為4與5的掉落
}

func (p *Action_HitCMD) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})

	if attackID, ok := m["AttackID"].(string); ok {
		p.AttackID = attackID
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse attackID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "Action_HitCMD")
		return false
	}
	if monsterIdxsInterface, ok := m["MonsterIdxs"].([]interface{}); ok {
		var monsterIdxs []int
		for _, idx := range monsterIdxsInterface {
			if floatIdx, ok := idx.(float64); ok {
				monsterIdxs = append(monsterIdxs, int(floatIdx))
			} else {
				log.WithFields(log.Fields{
					"invalidType":  reflect.TypeOf(idx),
					"invalidValue": idx,
					"log":          "parse MonsterIdxs資料錯誤",
				}).Errorf("%s Parse error: %s", logger.LOG_Pack, "Action_HitCMD")
				return false
			}
		}
		p.MonsterIdxs = monsterIdxs
	}

	// if monsterIdxs, ok := m["MonsterIdxs"].([]float64); ok {
	// 	p.MonsterIdxs = monsterIdxs
	// } else {
	// 	log.WithFields(log.Fields{
	// 		"log": "parse MonsterIdxs資料錯誤",
	// 	}).Errorf("%s Parse error: %s", logger.LOG_Pack, "Action_HitCMD")
	// 	return false
	// }

	if spellJsonID, ok := m["SpellJsonID"].(string); ok {
		p.SpellJsonID = spellJsonID
	} else {
		log.WithFields(log.Fields{
			"log": "parse SpellJsonID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "Action_HitCMD")
		return false
	}

	return true

}
