package packet

import (
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/setting"
	logger "matchgame/logger"
)

// 設定英雄
type Action_SetHeroCMD struct {
	CMDContent
	HeroID     int    // 玩家選擇英雄
	HeroSkinID string // 玩家英雄Skin
}

// 設定英雄回傳client
type Action_SetHeroCMD_Reply struct {
	CMDContent
	HeroIDs     [setting.PLAYER_NUMBER]int    // 玩家使用英雄ID清單
	HeroSkinIDs [setting.PLAYER_NUMBER]string // 玩家使用英雄SkinID清單
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
