package packet

import (
	"fmt"
	logger "matchmaker/Logger"

	log "github.com/sirupsen/logrus"
)

type CreateRoomCMD struct {
	CMDContent
	PlayerIDs []string
	CreaterID string
	MapID     string
}

func (cmd *CreateRoomCMD) Parse(content CMDContent) bool {
	m := content.(map[string]interface{})
	if playerIDs, ok := m["PlayerIDs"].([]interface{}); ok {
		cmd.PlayerIDs = make([]string, len(playerIDs))
		for i, v := range playerIDs {
			cmd.PlayerIDs[i] = fmt.Sprint(v)
		}
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "playerIDs資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Packet, "CreateRoomCMD")
		return false
	}

	if value, ok := m["CreaterID"].(string); ok {
		cmd.CreaterID = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "CreaterID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Packet, "CreateRoomCMD")
		return false
	}

	if value, ok := m["MapID"].(string); ok {
		cmd.MapID = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "MapID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Packet, "CreateRoomCMD")
		return false
	}

	return true
}
