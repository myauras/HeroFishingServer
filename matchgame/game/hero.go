package game

import (
	"fmt"
	"herofishingGoModule/gameJson"

	log "github.com/sirupsen/logrus"

	// "herofishingGoModule/utility"
	"matchgame/logger"
)

type Hero struct {
	ID             int                           // 英雄ID
	SkinID         string                        // SkinID
	Spells         [3]gameJson.HeroSpellJsonData // 英雄技能
	SpellLVs       [3]int                        // 英雄技能等級
	UsedSpellPoint int                           // 已使用的英雄技能點
}

// 取得英雄技能
func (hero *Hero) GetSpell(idx int) (gameJson.HeroSpellJsonData, error) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return gameJson.HeroSpellJsonData{}, fmt.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
	}
	return hero.Spells[(idx - 1)], nil // Spells索引是存0~2所以idx要-1
}
