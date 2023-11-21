package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)
// 帳號登入
type AuthCMD struct {
	CMDContent
	Token string
}
// 帳號登入回傳client
type AuthCMD_Reply struct {
	CMDContent
	IsAuth    bool   // 是否驗證成功
	ConnToken string // 連線Token
	Index     int32  // 玩家座位
}

func (p *AuthCMD) Parse(common CMDContent) bool {
	m := common.(map[string]interface{})
	if value, ok := m["Token"].(string); ok {
		p.Token = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "Token資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "AuthCMD")
		return false
	}
	return true
}
