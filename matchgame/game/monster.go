package game

import (
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/gameJson"
	"matchgame/logger"
)

type Monster struct {
	MonsterJson gameJson.MonsterJsonData // 怪物表Json
	MonsterIdx  int                      // 怪物唯一索引, 在怪物被Spawn後由server產生
	RouteJson   gameJson.RouteJsonData   // 路徑表Json
	SpawnTime   float64                  // 在遊戲時間第X秒時被產生的
}

func (monster *Monster) MonsterDie() {
	log.Infof("%s 怪物死亡 Index: %v ID: %s", logger.LOG_Monster, monster.MonsterIdx, monster.MonsterJson.ID)
}
