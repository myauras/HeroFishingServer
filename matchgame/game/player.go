package game

import (
	// log "github.com/sirupsen/logrus"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	// "matchgame/logger"
	gSetting "matchgame/setting"
	"time"
)

// 玩家
type Player struct {
	DBPlayer     *mongo.DBPlayer         // 玩家DB資料
	RedisPlayer  *redis.RedisPlayer      // RedisDB玩家實例
	Index        int                     // 玩家在房間的索引(座位)
	MyHero       *Hero                   // 使用中的英雄
	LastUpdateAt time.Time               // 上次收到玩家更新封包(心跳)
	ConnTCP      *gSetting.ConnectionTCP // TCP連線
	ConnUDP      *gSetting.ConnectionUDP // UDP連線
}

// 玩家點數增減
func (player *Player) AddPoint(value int64) {
	player.RedisPlayer.AddPoint(value)
	player.DBPlayer.Point += int64(value)
}

// 英雄經驗增減
func (player *Player) AddHeroExp(value int) {
	player.RedisPlayer.AddHeroExp(value)
	player.DBPlayer.HeroExp += int32(value)
}

// 將玩家連線斷掉
func (player *Player) CloseConnection() {
	if player == nil {
		return
	}
	if player.ConnTCP.Conn != nil {
		player.ConnTCP.Conn.Close()
		player.ConnTCP.Conn = nil
		player.ConnTCP = nil
	}
	if player.ConnUDP.Conn != nil {
		player.ConnUDP.Conn = nil
		player.ConnUDP = nil
	}
}
