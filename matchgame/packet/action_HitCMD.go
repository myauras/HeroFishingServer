package packet

import ()

// 命中怪物
type Action_HitCMD struct {
	CMDContent
	MonsterIDs  []int // 此次命中怪物ID清單
	SpellJsonID int   // 技能表ID
	AttackID    int   // 攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
}

// 命中怪物回傳client
type Action_HitCMD_Reply struct {
	CMDContent
	
}

func (p *Action_HitCMD) Parse(common CMDContent) bool {
	return true
}
