package packet

import (
	"herofishingGoModule/setting"
)

type Action_SetHeroCMD_Reply struct {
	CMDContent
	HeroIDs [setting.PLAYER_NUMBER]int32 // 玩家使用英雄ID清單
}
