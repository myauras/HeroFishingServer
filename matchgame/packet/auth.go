package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 帳號登入
type Auth struct {
	CMDContent
	Token string
}

// 帳號登入回傳client
type Auth_ToClient struct {
	CMDContent
	IsAuth    bool   // 是否驗證成功
	ConnToken string // 連線Token
	Index     int    // 玩家座位
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
