package packet

type UpdateGame_ToClient struct {
	CMDContent
	GameTime float64
}

type UpdateGame struct {
	CMDContent
}
