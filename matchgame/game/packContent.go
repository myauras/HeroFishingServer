package game

import (
	"herofishingGoModule/setting"
)

// 玩家狀態更新(TCP)
type UpdateRoomContent struct {
	PlayerStatuss [setting.PLAYER_NUMBER]PlayerStatus
}

// Server狀態更新(UDP)
type ServerStateContent struct {
	ServerTime float64
}
