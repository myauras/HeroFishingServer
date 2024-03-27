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
	skinID       string                        // 英雄JsonSkinID
	spells       [3]gameJson.HeroSpellJsonData // 英雄技能
	heroExp      int                           // 英雄經驗
	spellLVs     [3]int                        // 英雄技能等級, SpellLVs索引0到2(技能1到技能3), SpellLV值是0到4, 0是尚未學習,s 4是等級4
	spellCharges [3]int                        // 英雄技能充能, 索引0到2(技能1到技能3)
}

// 設定遊戲房內玩家使用英雄ID
func (hero *Hero) SetHero(heroID int, heroSkinID string) {
	if hero == nil {
		return
	}
	heroJson, err := gameJson.GetHeroByID(strconv.Itoa(heroID))
	if err != nil {
		log.Errorf("%s gameJson.GetHeroByID(strconv.Itoa(heroID))", logger.LOG_Room)
		return
	}
	spellJsons := heroJson.GetSpellJsons()
	hero.ID = heroID
	hero.skinID = heroSkinID
	hero.spells = spellJsons
}

// 取得英雄普攻JsonID
func (hero *Hero) GetAttackJsonID() string {
	spellJsonID := fmt.Sprintf("%s_attack", strconv.Itoa(hero.ID))
	return spellJsonID
}

func (hero *Hero) GetUsedSpellPoint() int {
	usedSpellPoint := 0
	for _, v := range hero.spellLVs {
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

// 取得英雄技能, 傳入0~2
func (hero *Hero) GetSpellJson(idx int) (gameJson.HeroSpellJsonData, error) {
	if idx < 0 || idx > 2 {
		log.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return gameJson.HeroSpellJsonData{}, fmt.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
	}
	return hero.spells[idx], nil
}

// 取得英雄技能等級, 傳入0~2
func (hero *Hero) GetSpellLV(idx int) int {
	if idx < 0 || idx > 2 {
		log.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return 0
	}
	return hero.spellLVs[idx]
}

// 設定英雄技能等級, idx傳入0~2
func (hero *Hero) AddSpellLV(idx int) {
	if idx < 0 || idx > 2 {
		log.Errorf("%s SetSpellLV傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return
	}
	hero.spellLVs[idx] += 1
}

// 取得英雄技能充能, 傳入0~2
func (hero *Hero) GetSpellCharge(idx int) int {
	if idx < 0 || idx > 2 {
		log.Errorf("%s GetSpell傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return 0
	}
	return hero.spellCharges[idx]
}

// 設定英雄技能充能, idx傳入0~2
func (hero *Hero) AddSpellChage(idx int, value int) {
	if idx < 0 || idx > 2 {
		log.Errorf("%s SetSpellLV傳入錯誤索引: %v", logger.LOG_Setting, idx)
		return
	}
	hero.spellCharges[idx] += value
}
