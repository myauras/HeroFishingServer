package packet

import (
	"herofishingGoModule/setting"
)

type UpdateGame_ToClient struct {
	CMDContent
	GameTime         float64                      // 遊戲開始X秒
	PlayerGainPoints [setting.PLAYER_NUMBER]int64 // 所有玩家的總獲得點數
}

type UpdateGame struct {
	CMDContent
}
