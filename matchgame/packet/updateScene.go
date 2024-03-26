package packet

import "herofishingGoModule/utility"

// "herofishingGoModule/setting"

type UpdateScene struct {
	CMDContent
}

type UpdateScene_ToClient struct {
	CMDContent
	Monsters     []PackMonster // 生怪清單(仍有效的生怪事件才傳, 如果該事件的怪物全數死亡就不用傳)
	SceneEffects []SceneEffect // 場景效果清單(還沒結束的效果 跟 永久影響的效果才需要傳)
}

type PackMonster struct {
	ID   int             // 怪物JsonID, JsonID
	Idx  int             // 怪物索引
	Pos  utility.Vector2 // 目前座標
	Type string          // 怪物類型
}

type SceneEffect struct {
	Name    string  // 效果名稱
	Value   float64 // 效果數值
	AtTime  float64 // 在遊戲時間第X秒觸發
	EndTime float64 // 在遊戲時間第X秒時結束
}
