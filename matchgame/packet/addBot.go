package packet

import (
// logger "matchgame/logger"
// log "github.com/sirupsen/logrus"
)

// 帳號登入
type AddBot struct {
	CMDContent
}

// 帳號登入回傳client
type AddBot_ToClient struct {
	CMDContent
	Success bool // 是否加入Bot成功
	Index   int  // Bot座位
}

func (p *AddBot) Parse(common CMDContent) bool {
	return true
}
