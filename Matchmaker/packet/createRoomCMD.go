package packet

import (
	logger "matchmaker/logger"

	log "github.com/sirupsen/logrus"
)

type CreateRoom struct {
	CMDContent
	CreaterID string
	DBMapID   string
}

type CreateRoom_ToClient struct {
	CMDContent
	CreaterID     string   // 創房者PlayerID
	PlayerIDs     []string // 房間內的所有PlayerID, 索引就是玩家的座位, 一進房間後就不會更動 PlayerIDs[0]就是在座位0玩家的PlayerID
	DBMapID       string   // DB地圖ID
	DBMatchgameID string   // DBMatchgame的ID(由Matchmaker產生，格視為[玩家ID]_[累加數字]_[日期時間])
	IP            string   // Matchmaker派發Matchgame的IP
	Port          int32    // Matchmaker派發Matchgame的Port
	PodName       string   // Matchmaker派發Matchgame的Pod名稱
}

func (cmd *CreateRoom) Parse(content CMDContent) bool {
	m := content.(map[string]interface{})
	if value, ok := m["CreaterID"].(string); ok {
		cmd.CreaterID = value
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "CreaterID資料錯誤",
		}).Errorf("%s Parse packet error: %s", logger.LOG_Pack, "CreateRoom")
		return false
	}

	if value, ok := m["DBMapID"].(string); ok {
		cmd.DBMapID = value

	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"log": "DBMapID資料錯誤",
		}).Errorf("%s Parse error: %s", logger.LOG_Pack, "CreateRoom")
		return false
	}

	return true
}
