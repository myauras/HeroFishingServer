package gamemath

import (
	"matchgame/logger"
	"testing"

	log "github.com/sirupsen/logrus"
)

var model = Model{
	GameRTP:        0.95, // 遊戲RTP
	SpellSharedRTP: 0.5,  // 分給技能掉落的RTP, 填0.5代表普攻有0.95-0.5=0.45的RTP而0.5被分給技能掉落
}

func TestGetSpellKP(t *testing.T) {

	p := model.GetAttackKP(100, 1, true)
	log.Infof("%s p: %v", logger.LOG_MathModel, p)
}
