package packet

// "herofishingGoModule/setting"

type MonsterDie_ToClient struct {
	CMDContent
	DieMonsters []DieMonster // 死亡怪物清單
}

// 死亡的怪物
type DieMonster struct {
	ID  int
	Idx int
}
