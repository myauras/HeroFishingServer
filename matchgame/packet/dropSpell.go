package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 使用道具
type DropSpell struct {
	CMDContent
	DropSpellJsonID int // DropSpell表ID
	// 其他使用道具需要的參數
	AttackID int
}

// 使用道具回傳client
type DropSpell_ToClient struct {
	CMDContent
	Success         bool
	PlayerIdx       int // 玩家座位索引
	DropSpellJsonID int // DropSpell表ID
}

func (dropSpell *DropSpell) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})

	if dropSpellJsonID, ok := m["DropSpellJsonID"].(float64); ok {
		dropSpell.DropSpellJsonID = int(dropSpellJsonID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse DropSpellJsonID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "DropSpellJsonID")
		return false
	}

	if attackID, ok := m["AttackID"].(float64); ok {
		dropSpell.AttackID = int(attackID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse AttackID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "AttackID")
		return false
	}

	return true

}
