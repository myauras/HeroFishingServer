package packet

import ()

type UpdateGame_ToClient struct {
	CMDContent
	GameTime float64 // 遊戲開始X秒
}

type UpdateGame struct {
	CMDContent
}
