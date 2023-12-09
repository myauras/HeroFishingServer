package setting

// 時間參數
const (
	DISCONNECT_CHECK_INTERVAL_SECS = 60 // 每X秒做玩家斷線檢測
)

// 房間參數
const (
	RETRY_CREATE_GAMESERVER_TIMES = 2 // 開房失敗時重試X次
	RETRY_INTERVAL_SECONDS        = 1 // 開房失敗重試間隔X秒
	ROUTINE_CHECK_OCCUPIED_ROOM   = 5 // 每X分鐘檢查佔用房間
)
