package packet

type AuthCMD_Reply struct {
	CMDContent
	IsAuth    bool   // 是否驗證成功
	ConnToken string // 連線Token
	Index     int32  // 玩家座位
}
