package packet

type AuthCMD_Reply struct {
	CMDContent
	IsAuth    bool
	ConnToken string
}
