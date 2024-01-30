package redis

import "encoding/json"

// 命令類型
const (
	CMD_PLAYERLEFT  = "PLAYERLEFT"
	CMD_GAMECREATED = "GAMECREATED"
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
type GameCreated struct {
	MatchgameID string `json:"PlayerID"` // 遊戲房ID
	PackID      int    `json:"PackID"`   // Matchmaker要回送CreateRoom_ToClient封包時的PackID
}
