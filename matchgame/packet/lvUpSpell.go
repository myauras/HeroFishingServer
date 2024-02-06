package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type LvUpSpell_ToClient struct {
	CMDContent
	Success bool
}

type LvUpSpell struct {
	CMDContent
	SpellIdx int
}

func (lvUpSpell *LvUpSpell) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})

	if spellIdx, ok := m["SpellIdx"].(float64); ok {
		lvUpSpell.SpellIdx = int(spellIdx)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse lvUpSpell資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "SpellIdx")
		return false
	}

	return true

}
