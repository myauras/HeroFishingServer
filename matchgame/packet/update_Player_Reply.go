package packet

import (
	"herofishingGoModule/setting"
	gSetting "matchgame/setting"
)

type Update_Player_Reply struct {
	CMDContent
	Players [setting.PLAYER_NUMBER]*gSetting.Player // 玩家陣列

}
