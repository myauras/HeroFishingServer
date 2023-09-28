package setting

// 伺服器設定
const (
	PLAYER_NUMBER                  = 4   // 遊戲房最多X位玩家
	TIME_UPDATE_INTERVAL_MS        = 200 // 每X毫秒更新Server時間
	AGONES_HEALTH_PIN_INTERVAL_SEC = 2   //每X秒檢查AgonesServer是否正常運作(官方文件範例是用2秒)
)
