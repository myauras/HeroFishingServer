package redis

import "encoding/json"

// 命令類型-Matchgame to Matchmaker
const (
	CMD_PLAYERLEFT  = "PLAYERLEFT"
	CMD_GAMECREATED = "GAMECREATED"
)

// 命令類型-Matchmaker to Matchgame
const (
	CMD_KICK_DISCONNECTED_PLAYER = "KICK_DISCONNECTED_PLAYER"
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
type KickDisconnectedPlayer struct {
	PlayerID string `json:"PlayerID"` // 玩家ID
}
