package packet

import (
	logger "matchmaker/logger"

	log "github.com/sirupsen/logrus"
)

type Auth struct {
	CMDContent
	Token string
}
type AuthC_ToClient struct {
	CMDContent
	IsAuth bool
}

func (p *Auth) Parse(common CMDContent) bool {
	m := common.(map[string]interface{})
	if value, ok := m["Token"].(string); ok {
		p.Token = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "Token資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "Auth")
		return false
	}
	return true
}
