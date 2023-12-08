package setting

import (
	"encoding/json"
	"net"
)

// 伺服器設定
const (
	TIME_UPDATE_INTERVAL_MS        = 1000 // 每X毫秒送UPDATEGAME_TOCLIENT封包給client(心跳檢測)
	AGONES_HEALTH_PIN_INTERVAL_SEC = 1    // 每X秒檢查AgonesServer是否正常運作(官方文件範例是用2秒)
)


type ConnectionTCP struct {
	Conn    net.Conn      // TCP連線
	Encoder *json.Encoder // 連線編碼
	Decoder *json.Decoder // 連線解碼
}
type ConnectionUDP struct {
	Conn      net.PacketConn // UDP連線
	Addr      net.Addr       // 玩家連線地址
	ConnToken string         // 驗證Token
}
