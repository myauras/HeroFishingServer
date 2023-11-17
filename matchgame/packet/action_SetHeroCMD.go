package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type Action_SetHeroCMD struct {
	CMDContent
	HeroID     int    // 玩家選擇英雄
	HeroSkinID string // 玩家英雄Skin
}

func (p *Action_SetHeroCMD) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})

	if heroID, ok := m["HeroID"].(float64); ok {
		p.HeroID = int(heroID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse heroID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "PAction_SetHeroCMD")
		return false
	}

	if heroSkinID, ok := m["HeroSkinID"].(string); ok {
		p.HeroSkinID = heroSkinID
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse heroSkinID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "PAction_SetHeroCMD")
		return false
	}
	return true
}
