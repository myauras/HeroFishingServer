package packet

type CreateRoomCMD_Reply struct {
	CMDContent
	PlayerIDs      []string
	DBMapID        string
	GameServerIP   string
	GameServerPort int32
	GameServerName string
}
