package packet

// "herofishingGoModule/setting"

type UpdateScene struct {
	CMDContent
}

type UpdateScene_ToClient struct {
	CMDContent
	Spawns       []Spawn       // 生怪清單(仍有效的生怪事件才傳, 如果該事件的怪物全數死亡就不用傳)
	SceneEffects []SceneEffect // 場景效果清單(還沒結束的效果 跟 永久影響的效果才需要傳)
}

type Spawn struct {
	CMDContent
	RouteJsonID int        // 路徑JsonID
	SpawnTime   float64    // 在遊戲時間第X秒時被產生的
	IsBoss      bool       // 是否為Boss生怪
	Monsters    []*Monster // 怪物清單
}

type Monster struct {
	JsonID  int             // 怪物JsonID
	Idx     int             // 怪物索引
	Death   bool            // 是否已死亡
	Effects []MonsterEffect // 怪物效果清單(還沒結束的效果 跟 永久影響的效果才需要傳)
}

type MonsterEffect struct {
	Name     string  // 效果名稱
	AtTime   float64 // 在遊戲時間第X秒觸發
	Duration float64 // 效果持續X秒
}
type SceneEffect struct {
	Name     string  // 效果名稱
	Value    float64 // 效果數值
	AtTime   float64 // 在遊戲時間第X秒觸發
	Duration float64 // 效果持續X秒
}
