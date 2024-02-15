package packet

import (
	logger "matchgame/logger"
	"reflect"

	log "github.com/sirupsen/logrus"
)

// 命中怪物
type Hit struct {
	CMDContent
	AttackID    int    // 攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	MonsterIdxs []int  // 此次命中怪物索引清單
	SpellJsonID string // 技能表ID
}

// 命中怪物回傳client
type Hit_ToClient struct {
	CMDContent
	PlayerIdx        int     // 玩家座位索引
	KillMonsterIdxs  []int   // 擊殺怪物索引清單, [1,1,3]就是依次擊殺索引為1,1與3的怪物
	GainPoints       []int64 // 獲得點數清單, [1,1,3]就是依次獲得點數1,1與3
	GainHeroExps     []int32 // 獲得英雄經驗清單, [1,1,3]就是依次獲得英雄經驗1,1與3
	GainSpellCharges []int32 // 獲得技能充	能清單, [1,1,3]就是依次獲得技能1,技能1,技能3的充能
	GainDrops        []int32 // 獲得掉落清單, [1,1,3]就是依次獲得DropJson中ID為1,1與3的掉落
	PTBuffer         int64   // 溢位的Point
}

func (hit *Hit) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})

	if attackID, ok := m["AttackID"].(float64); ok {
		hit.AttackID = int(attackID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse attackID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Hit")
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
				}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Hit")
				return false
			}
		}
		hit.MonsterIdxs = monsterIdxs
	}

	if spellJsonID, ok := m["SpellJsonID"].(string); ok {
		hit.SpellJsonID = spellJsonID
	} else {
		log.WithFields(log.Fields{
			"log": "parse SpellJsonID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Hit")
		return false
	}

	return true

}
