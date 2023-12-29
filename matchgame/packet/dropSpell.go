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

	return true

}
