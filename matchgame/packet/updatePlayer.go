package packet

import (
	"herofishingGoModule/setting"
	gSetting "matchgame/setting"
)

type UpdatePlayer_ToClient struct {
	CMDContent
	Players [setting.PLAYER_NUMBER]*gSetting.Player // 玩家陣列

}
