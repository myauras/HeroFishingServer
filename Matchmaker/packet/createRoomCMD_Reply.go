package packet

type CreateRoomCMD_Reply struct {
	CMDContent
	CreaterID     string   // 創房者PlayerID
	PlayerIDs     []string // 房間內的所有PlayerID
	DBMapID       string   // DB地圖ID
	DBMatchgameID string   // DBMatchgame的ID(由Matchmaker產生，格視為[玩家ID]_[累加數字]_[日期時間])
	IP            string   // Matchmaker派發Matchgame的IP
	Port          int32    // Matchmaker派發Matchgame的Port
	PodName       string   // Matchmaker派發Matchgame的Pod名稱
}
