package gamemath

// log "github.com/sirupsen/logrus"
// "matchgame/logger"

// 擊中(包含普攻, 英雄技能, 道具技能, 互動物件等任何攻擊)
type Hits struct {
	// 攻擊ID格式為 [玩家房間index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號, 擊中波次是server端累加的流水號)
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	ID          string  // 攻擊ID
	ExpiredTime float64 // 過期時間, 房間中的GameTime超過此值就會視為此技能已經結束
	MonsterIdxs [][]int // [波次]-[擊中怪物索引清單]

	// 以下為技能表參數
	RTP   float64 // 技能RTP
	Waves int     // 技能總共波次
	Hits  int     // 技能每波命中最大怪物數量
}
// 擊中結果
type HitsResult struct {
	
}
