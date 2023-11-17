package setting

import (
	"encoding/json"
	"net"
)

// 玩家
type Player struct {
	ID       string
	Index    int32 // 玩家在房間的索引(座位)
	Status   *PlayerStatus
	LeftSecs float64       // 玩家已離開遊戲房X秒
	ConnTCP  ConnectionTCP // TCP連線
	ConnUDP  net.Conn      // UDP連線
}

// 將玩家連線斷掉
func (player *Player) CloseConnection() {
	if player == nil {
		return
	}
	if player.ConnTCP.Conn != nil {
		player.ConnTCP.Conn.Close()
		player.ConnTCP.Conn = nil
	}
	if player.ConnUDP != nil {
		player.ConnUDP.Close()
		player.ConnUDP = nil
	}
}

type ConnectionTCP struct {
	Conn    net.Conn      // TCP連線
	Encoder *json.Encoder // 連線編碼
	Decoder *json.Decoder // 連線解碼
}

// 玩家狀態
type PlayerStatus struct {
}

// 伺服器設定
const (
	TIME_UPDATE_INTERVAL_MS        = 200 // 每X毫秒更新Server時間
	AGONES_HEALTH_PIN_INTERVAL_SEC = 2   //每X秒檢查AgonesServer是否正常運作(官方文件範例是用2秒)
)
