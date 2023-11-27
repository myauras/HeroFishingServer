package packet

type Spawn_ToClient struct {
	CMDContent
	IsBoss      bool    // 是否為Boss生怪
	MonsterIDs  []int   // 怪物JsonIDs
	MonsterIdxs []int   // 怪物唯一索引清單
	RouteID     int     // 路徑JsonID
	SpawnTime   float64 // 在遊戲時間第X秒時被產生的
}
