package game

import (
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	// "matchgame/logger"
	// log "github.com/sirupsen/logrus"
)

type Monster struct {
	MonsterIdx int                    // 怪物唯一索引, 在怪物被Spawn後由server產生
	RouteJson  gameJson.RouteJsonData //
	ID         int                    // 怪物JsonID
	EXP        int                    // 怪物經驗
	Odds       int                    // 怪物賠率
	DropID     int                    // 怪物掉落ID
	DropRTP    int                    // 怪物掉落RTP
	SpawnPos   utility.Vector2        // 出生座標
	SpawnTime  float64                // 在遊戲時間第X秒時被產生的
	LeaveTime  float64                // 在遊戲時間第X秒時要被移除
}

// func (monster *Monster) GetCurPos() {
// 	passTime := MyRoom.GameTime - monster.SpawnTime
// }
