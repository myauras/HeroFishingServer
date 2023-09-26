package game

import (
	"encoding/json"
	"net"
	"time"
)

// 玩家
type Player struct {
	ID       string
	Status   *PlayerStatus
	Conn_TCP net.Conn
	Conn_UDP net.Conn
	Encoder  *json.Encoder
	Decoder  *json.Decoder
}

// 玩家狀態
type PlayerStatus struct {
}

func (player *Player) CloseConnection() {
	if player.Conn_TCP != nil {
		player.Conn_TCP.Close()
		player.Conn_TCP = nil
	}
	if player.Conn_UDP != nil {
		player.Conn_UDP.Close()
		player.Conn_UDP = nil
	}
}
