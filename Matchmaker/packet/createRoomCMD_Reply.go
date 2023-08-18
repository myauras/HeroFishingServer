package packet

type CreateRoomCMD_Reply struct {
	CMDContent
	PlayerIDs      []string
	MapID          string
	GameServerIP   string
	GameServerPort int32
	GameServerName string
}
