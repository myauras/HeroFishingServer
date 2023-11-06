package packet

import ()

type SpawnCMD struct {
	CMDContent
	MonsterID int     // 怪物JsonID
	RouteID   int     // 路徑JsonID
	SpawnTime float64 // 在遊戲時間第X秒時被產生的
}
