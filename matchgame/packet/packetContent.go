package packet

import (
	"matchgame/game"
)

type UpdateRoomContent struct {
	PlayerStatuss [game.PLAYER_NUMBER]game.PlayerStatus
}
