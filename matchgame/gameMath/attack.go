package gamemath

// log "github.com/sirupsen/logrus"
// "matchgame/logger"

// 攻擊(包含普攻, 英雄技能, 道具技能, 互動物件等任何攻擊)事件
type AttackEvent struct {
	// 攻擊AttackID格式為 [玩家房間index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	AttackID    string  // 攻擊ID
	ExpiredTime float64 // 過期時間, 房間中的GameTime超過此值就會視為此技能已經結束
	MonsterIdxs [][]int // [波次]-[擊中怪物索引清單]

	// 以下為技能表參數
	SpellJSonRTP   float64 // 技能RTP
	SpellJsonWaves int     // 技能總共波次
	SpellJsonHits  int     // 技能每波命中最大怪物數量
}

// 攻擊結果
type AttackResult struct {
}
