package game

import (
	"herofishingGoModule/gameJson"
	// "matchgame/logger"
	// log "github.com/sirupsen/logrus"
)

type Monster struct {
	MonsterJson gameJson.MonsterJsonData // 怪物表Json
	MonsterIdx  int                      // 怪物唯一索引, 在怪物被Spawn後由server產生
	RouteJson   gameJson.RouteJsonData   //
	SpawnTime   float64                  // 在遊戲時間第X秒時被產生的
	LeaveTime   float64                  // 在遊戲時間第X秒時要被移除
}
