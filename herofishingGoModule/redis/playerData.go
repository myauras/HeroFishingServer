package redis

import (
	"context"
	"fmt"
	"time"

	logger "herofishingGoModule/logger"

	"github.com/mitchellh/mapstructure"
	redis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

var dbWriteMinMiliSecs = 1000
var rdb *redis.Client
var ctx context.Context
var cancel context.CancelFunc
var players map[string]*RedisPlayer

type RedisPlayer struct {
	id             string // Redis的PlayerID是"player-"+mongodb player id, 例如player-6538c6f219a12eb9e4ded943
	pointChan      chan int64
	heroExpChan    chan int
	pointBalance   int64 //暫存點數修改
	heroExpBalance int   //暫存經驗修改
}
type DBPlayer struct {
	ID      string
	Point   chan int64
	HeroExp chan int
}

// 將暫存的數據寫入RedisDB
func (player *RedisPlayer) WritePlayerUpdateToRedis() {

	if player.pointBalance != 0 {
		_, err := rdb.HIncrBy(ctx, player.id, "point", int64(player.pointBalance)).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis錯誤: %v", logger.LOG_Redis, err)
		}
		player.pointBalance = 0
	}
	if player.heroExpBalance != 0 {
		_, err := rdb.HIncrBy(ctx, player.id, "heroExp", int64(player.heroExpBalance)).Result() // 轉換為 int64
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis錯誤: %v", logger.LOG_Redis, err)
		}
		player.heroExpBalance = 0
	}
}

// 關閉玩家channel
func (player *RedisPlayer) closeChannel() {
	player.WritePlayerUpdateToRedis()
	close(player.pointChan)
	close(player.heroExpChan)
}

// 初始化RedisDB, 已經初始化過會直接return
func Init() {
	if rdb != nil {
		return
	}
	rdb = redis.NewClient(&redis.Options{
		Addr:     "redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com:10238",
		Password: "dMfmpIDd0BTIyeCnOkBhuznVPxd7V7yx",
		DB:       0,
	})
	ctx, cancel = context.WithCancel(context.Background())
	players = make(map[string]*RedisPlayer)
}
func Ping() error {
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return err
	}
	return nil
}

// 關閉Redis
func CloseAll() {
	cancel()
	for _, p := range players {
		p.closeChannel()
	}
}

// 關閉玩家
func ClosePlayer(playerID string) {
	if _, ok := players[playerID]; ok {
		players[playerID].closeChannel()
		delete(players, playerID) // 從 map 中移除
	} else {
		log.Errorf("%s ClosePlayer錯誤 玩家 %s 不存在map中", logger.LOG_Redis, playerID)
		return
	}
}

// 關閉玩家
func (player *RedisPlayer) ClosePlayer() {
	ClosePlayer(player.id)
}

// 建立玩家資料
func CreatePlayerData(playerID string, point int, heroExp int) (*RedisPlayer, error) {
	playerID = "player-" + playerID

	dbPlayer, err := GetPlayerDBData(playerID)
	if err != nil || dbPlayer.ID == "" {
		// 建立玩家RedisDB資料
		_, err := rdb.HMSet(ctx, playerID, map[string]interface{}{
			"id":      playerID,
			"point":   point,
			"heroExp": heroExp,
		}).Result()
		if err != nil {
			return nil, fmt.Errorf("%s createPlayerData錯誤: %v", logger.LOG_Redis, err)
		}
	}

	player := RedisPlayer{
		id:          playerID,
		pointChan:   make(chan int64),
		heroExpChan: make(chan int),
	}

	if _, ok := players[playerID]; !ok {
		players[playerID] = &player
	} else {
		return nil, fmt.Errorf("%s createPlayerData錯誤 玩家 %s 已存在map中", logger.LOG_Redis, playerID)
	}
	go updatePlayer(&player)
	return &player, nil
}

// 增加點數
func (player *RedisPlayer) AddPoint(value int64) {
	player.pointChan <- value
}

// 增加英雄經驗
func (player *RedisPlayer) AddHeroExp(value int) {
	player.heroExpChan <- value
}

// 暫存資料寫入並每X毫秒更新上RedisDB
func updatePlayer(player *RedisPlayer) {
	ticker := time.NewTicker(time.Duration(dbWriteMinMiliSecs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case pointChange := <-player.pointChan:
			player.pointBalance += pointChange
		case heroExpChange := <-player.heroExpChan:
			player.heroExpBalance += heroExpChange
		case <-ticker.C:
			player.WritePlayerUpdateToRedis()
		case <-ctx.Done():
			return
		}
	}
}

// 取得RedisDB中Player資料
func (player *RedisPlayer) GetPlayerDBData() {
	GetPlayerDBData(player.id)
}

// 取得RedisDB中Player資料, 找不到玩家資料時DBPlayer會返回0值
func GetPlayerDBData(playerID string) (DBPlayer, error) {
	var player DBPlayer
	val, err := rdb.HGetAll(ctx, playerID).Result()
	if err != nil {
		return player, fmt.Errorf("ShowPlayer錯誤: %v", err)
	}
	if len(val) == 0 { // 找不到資料回傳0值
		return player, nil
	}
	err = mapstructure.Decode(val, &player)
	if err != nil {
		return player, fmt.Errorf("RedisDB Plaeyr 反序列化錯誤: %v", err)
	}
	// log.Infof("%s playerID: %s point: %d heroExp: %d\n", logger.LOG_Redis, player.ID, player.Point, player.HeroExp)
	return player, nil

}
