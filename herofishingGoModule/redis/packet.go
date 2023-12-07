package redis

import ()

// 命令類型
const (
	CMD_PLAYERLEFT = "PLAYERLEFT"
)

type CMDContent interface {
}
type RedisPubSubPack struct {
	CMD     string
	Content CMDContent
}
type PlayerLeft struct {
	CMDContent
	PlayerID string // 玩家ID
}
