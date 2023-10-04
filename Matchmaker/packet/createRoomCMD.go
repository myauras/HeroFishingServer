package packet

import (
	logger "matchmaker/logger"

	log "github.com/sirupsen/logrus"
)

type CreateRoomCMD struct {
	CMDContent
	CreaterID string
	DBMapID   string
}

func (cmd *CreateRoomCMD) Parse(content CMDContent) bool {
	m := content.(map[string]interface{})
	if value, ok := m["CreaterID"].(string); ok {
		cmd.CreaterID = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "CreaterID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "CreateRoomCMD")
		return false
	}

	if value, ok := m["DBMapID"].(string); ok {
		cmd.DBMapID = value

	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "DBMapID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "CreateRoomCMD")
		return false
	}

	return true
}
