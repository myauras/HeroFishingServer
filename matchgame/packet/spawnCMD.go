package packet

type SpawnCMD struct {
	CMDContent
	IsBoss     bool    // 是否為Boss生怪
	MonsterIDs []int   // 怪物JsonIDs
	RouteID    int     // 路徑JsonID
	SpawnTime  float64 // 在遊戲時間第X秒時被產生的
}
