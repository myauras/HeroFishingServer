package packet

import (
	log "github.com/sirupsen/logrus"
	logger "matchgame/logger"
)

// 使用道具
type DropUse struct {
	CMDContent
	DropJsonID int // Drop表ID
	// 其他使用道具需要的參數
}

// 使用道具回傳client
type DropUse_ToClient struct {
	CMDContent
	PlayerIdx  int // 玩家座位
	DropJsonID int // Drop表ID
	// 其他使用道具需要的參數
}

func (dropUse *DropUse) Parse(common CMDContent) bool {

	m := common.(map[string]interface{})

	if dropJsonID, ok := m["DropJsonID"].(float64); ok {
		dropUse.DropJsonID = int(dropJsonID)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "parse DropJsonID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "DropUse")
		return false
	}

	return true

}
