package gamemath

import (
	"matchgame/logger"
	"testing"

	log "github.com/sirupsen/logrus"
)

var model = Model{
	GameRTP:   0.95,
	AttackRTP: 0.945,
}

func TestGetAttackKP(t *testing.T) {

	p := model.GetAttackKP(100)
	log.Infof("%s p: %v", logger.LOG_MathModel, p)
}
func TestGetSpellKP(t *testing.T) {

	p := model.GetSpellKP(100, 100)
	log.Infof("%s p: %v", logger.LOG_MathModel, p)
}

func TestHit(t *testing.T) {
	hit := Hits{
		ID:          "0_1",
		ExpiredTime: 10,
		MonsterIdxs: make([][]int, 3), // 波次為3的攻擊
		RTP:         100,
		Waves:       3,
		Hits:        5,
	}

	p := model.GetSpellKP(100, 100)
	log.Infof("%s p: %v", logger.LOG_MathModel, p)
}
