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
	SpellLVs       [4]int                        // 英雄技能等級, SpellLVs索引只使用1到3(技能1到技能3), SpellLV值是0到3, 0是尚未學習,s 3是等級3
	UsedSpellPoint int32                         // 已使用的英雄技能點
}

// 取得英雄技能
func (hero *Hero) GetSpell(idx int32) (gameJson.HeroSpellJsonData, error) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return gameJson.HeroSpellJsonData{}, fmt.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
	}
	return hero.Spells[(idx - 1)], nil // Spells索引是存0~2所以idx要-1
}
