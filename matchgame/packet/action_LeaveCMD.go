package packet

import ()

type Action_LeaveCMD struct {
	CMDContent
}
func (p *Action_LeaveCMD) Parse(common CMDContent) bool {
	return true
}
