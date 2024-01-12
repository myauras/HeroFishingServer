package packet

type Leave struct {
	CMDContent
}

type Leave_ToClient struct {
	CMDContent
	PlayerIdx int // 玩家座位索引
}

func (p *Leave) Parse(common CMDContent) bool {
	return true
}
