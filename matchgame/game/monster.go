package game

import (
	"herofishingGoModule/gameJson"
	// "matchgame/logger"
	// log "github.com/sirupsen/logrus"
)

type Monster struct {
	MonsterJson gameJson.MonsterJsonData // 怪物表Json
	MonsterIdx  int                      // 怪物唯一索引, 在怪物被Spawn後由server產生
	RouteJson   gameJson.RouteJsonData   // 路徑表Json
	SpawnTime   float64                  // 在遊戲時間第X秒時被產生的
	LeaveTime   float64                  // 在遊戲時間第X秒時要被移除
}

// 移除怪物
func (monster *Monster) RemoveMonster() {
	// log.Infof("%s 移除怪物 Idx: %v ID: %s", logger.LOG_Monster, monster.MonsterIdx, monster.MonsterJson.ID)
}
