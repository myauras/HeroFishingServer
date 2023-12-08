package packet

import (
// "herofishingGoModule/setting"
)

type UpdatePlayer_ToClient struct {
	CMDContent
	Players [4]*Player // 玩家陣列

}
type Player struct {
	ID    string
	Index int
}
