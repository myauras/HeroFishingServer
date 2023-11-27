package packet

type Leave struct {
	CMDContent
}

func (p *Leave) Parse(common CMDContent) bool {
	return true
}
