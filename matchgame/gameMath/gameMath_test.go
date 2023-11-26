package gamemath

import (
	"matchgame/logger"
	"testing"

	log "github.com/sirupsen/logrus"
)

var model = Model{
	GameRTP:        0.95,
	SpellSharedRTP: 0.945,
}

func TestGetSpellKP(t *testing.T) {

	p := model.GetAttackKP(100, 1, true)
	log.Infof("%s p: %v", logger.LOG_MathModel, p)
}
