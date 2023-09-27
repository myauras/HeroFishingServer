package game

import (
	"encoding/json"
	"net"
)

// 玩家
type Player struct {
	ID       string
	Index    int // 玩家在房間的索引(座位)
	Status   *PlayerStatus
	LeftSecs float64       // 玩家已離開遊戲房X秒
	ConnTCP  ConnectionTCP // TCP連線
}
type ConnectionTCP struct {
	Conn    net.Conn      // TCP連線
	Encoder *json.Encoder // 連線編碼
	Decoder *json.Decoder // 連線解碼
}

// 玩家狀態
type PlayerStatus struct {
}

func (player *Player) CloseConnection() {
	if player.ConnTCP.Conn != nil {
		player.ConnTCP.Conn.Close()
		player.ConnTCP.Conn = nil
	}
}
