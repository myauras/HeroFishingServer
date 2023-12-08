package redis

import "encoding/json"

// 命令類型
const (
	CMD_PLAYERLEFT = "PLAYERLEFT"
)

type CMDContent interface {
}
type RedisPubSubPack struct {
	CMD     string          `json:"CMD"`
	Content json.RawMessage `json:"Content"`
}
type PlayerLeft struct {
	PlayerID string `json:"PlayerID"` // 玩家ID
}
