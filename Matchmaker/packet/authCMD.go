package packet

import (
	logger "matchmaker/Logger"

	log "github.com/sirupsen/logrus"
)

type AuthCMD struct {
	CMDContent
	Token string
}

func (p *AuthCMD) Parse(common CMDContent) bool {
	m := common.(map[string]interface{})
	if value, ok := m["Token"].(string); ok {
		p.Token = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "Token資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Packet, "AuthCMD")
		return false
	}
	return true
}
