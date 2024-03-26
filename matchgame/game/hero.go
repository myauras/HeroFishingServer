package game

import (
	"fmt"
	"herofishingGoModule/gameJson"
	"strconv"

	log "github.com/sirupsen/logrus"

	// "herofishingGoModule/utility"
	"matchgame/logger"
)

type Hero struct {
	ID           int                           // 英雄JsonID
	SkinID       string                        // 英雄JsonSkinID
	Spells       [3]gameJson.HeroSpellJsonData // 英雄技能
	HeroExp      int32                         // 英雄經驗
	SpellLVs     [4]int                        // 英雄技能等級, SpellLVs索引只使用1到3(技能1到技能3), SpellLV值是0到3, 0是尚未學習,s 3是等級3
	SpellCharges [3]int32                      // 英雄技能充能
}

// 取得英雄普攻JsonID
func (hero *Hero) GetAttackJsonID() string {
	spellJsonID := fmt.Sprintf("%s_attack", strconv.Itoa(hero.ID))
	return spellJsonID
}

func (hero *Hero) GetUsedSpellPoint() int {
	usedSpellPoint := 0
	for _, v := range hero.SpellLVs {
		usedSpellPoint += v
	}
	return usedSpellPoint
}

// 取得英雄普攻Json
func (hero *Hero) GetAttackJson() (gameJson.HeroSpellJsonData, error) {
	jsonID := hero.GetAttackJsonID()
	spellJson, err := gameJson.GetHeroSpellByID(jsonID)
	if err != nil {
		errStr := fmt.Sprintf("GetAttackJson()時gameJson.GetHeroSpellByID(hitCMD.SpellJsonID) SpellJsonID: %s 錯誤: %v", jsonID, err)
		log.Errorf("%s %s", logger.LOG_Action, errStr)
		return gameJson.HeroSpellJsonData{}, err
	}
	return spellJson, nil
}

// 取得英雄技能
func (hero *Hero) GetSpell(idx int32) (gameJson.HeroSpellJsonData, error) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return gameJson.HeroSpellJsonData{}, fmt.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
	}
	return hero.Spells[(idx - 1)], nil // Spells索引是存0~2所以idx要-1
}
