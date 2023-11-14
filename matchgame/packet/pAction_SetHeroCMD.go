package packet

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

type PAction_SetHeroCMD struct {
	CMDContent
	Index  int32 // 玩家的座位索引
	HeroID int32 // 玩家選擇英雄
}

func (p *PAction_SetHeroCMD) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})
	if index, ok := m["Index"].(int32); ok {
		p.Index = index
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse index資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "PAction_SetHeroCMD")
		return false
	}

	if heroID, ok := m["HeroID"].(int32); ok {
		p.HeroID = heroID
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse heroID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "PAction_SetHeroCMD")
		return false
	}
	return true
}
