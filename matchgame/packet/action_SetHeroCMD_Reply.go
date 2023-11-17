package packet

import (
	"herofishingGoModule/setting"
)

type Action_SetHeroCMD_Reply struct {
	CMDContent
	HeroIDs     [setting.PLAYER_NUMBER]int  // 玩家使用英雄ID清單
	HeroSkinIDs [setting.PLAYER_NUMBER]string // 玩家使用英雄SkinID清單
}
