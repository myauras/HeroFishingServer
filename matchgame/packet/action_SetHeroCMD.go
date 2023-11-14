package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type Action_SetHeroCMD struct {
	CMDContent
	Index  int32 // 玩家的座位索引
	HeroID int32 // 玩家選擇英雄
}

func (p *Action_SetHeroCMD) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})
	if index, ok := m["Index"].(float64); ok {
		p.Index = int32(index)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse index資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "Action_SetHeroCMD")
		return false
	}

	if heroID, ok := m["HeroID"].(float64); ok {
		p.HeroID = int32(heroID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse heroID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "PAction_SetHeroCMD")
		return false
	}
	return true
}
