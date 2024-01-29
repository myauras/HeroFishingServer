package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type Auto struct {
	CMDContent
	IsAuto bool
}

type Auto_ToClient struct {
	CMDContent
	IsAuto bool
}

func (a *Auto) Parse(common CMDContent) bool {
	m := common.(map[string]interface{})
	if isAuto, ok := m["IsAuto"].(bool); ok {
		a.IsAuto = isAuto
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse IsAuto資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Auto")
		return false
	}
	return true
}
