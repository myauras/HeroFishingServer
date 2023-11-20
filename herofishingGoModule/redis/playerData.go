package redis

import (
	"context"
	"strconv"
	"time"

	logger "herofishingGoModule/logger"

	redis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

var dbWriteMinMiliSecs = 1000
var rdb *redis.Client
var ctx context.Context
var cancel context.CancelFunc
var players map[string]*redisPlayer

type redisPlayer struct {
	id          string // Redis的PlayerID是"player-"+mongodb player id, 例如player-6538c6f219a12eb9e4ded943
	goldChan    chan int
	expChan     chan int
	goldBalance int //暫存金幣修改
	expBalance  int //暫存經驗修改
}

// 將暫存的數據寫入RedisDB
func (player *redisPlayer) writePlayerUpdateToRedis() {

	if player.goldBalance != 0 {
		_, err := rdb.HIncrBy(ctx, player.id, "gold", int64(player.goldBalance)).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis錯誤: %v", logger.LOG_Redis, err)
		}
		player.goldBalance = 0
	}
	if player.expBalance != 0 {
		_, err := rdb.HIncrBy(ctx, player.id, "exp", int64(player.expBalance)).Result() // 轉換為 int64
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis錯誤: %v", logger.LOG_Redis, err)
		}
		player.expBalance = 0
	}
}

// 關閉玩家channel
func (player *redisPlayer) closeChannel() {
	player.writePlayerUpdateToRedis()
	close(player.goldChan)
	close(player.expChan)
}

// 初始化
func Init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com:10238",
		Password: "dMfmpIDd0BTIyeCnOkBhuznVPxd7V7yx",
		DB:       0,
	})
	ctx, cancel = context.WithCancel(context.Background())
	players = make(map[string]*redisPlayer)
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

// 建立玩家資料
func CreatePlayerData(playerID string, gold int, heroExp int) {
	playerID = "player-" + playerID
	_, err := rdb.HMSet(ctx, playerID, map[string]interface{}{
		"id":      playerID,
		"gold":    gold,
		"heroExp": heroExp,
	}).Result()
	if err != nil {
		log.Errorf("%s createPlayerData錯誤: %v", logger.LOG_Redis, err)
	}
	player := redisPlayer{
		id:       playerID,
		goldChan: make(chan int),
		expChan:  make(chan int),
	}
	if _, ok := players[playerID]; !ok {
		players[playerID] = &player
	} else {
		log.Errorf("%s createPlayerData錯誤 玩家 %s 已存在map中", logger.LOG_Redis, playerID)
		return
	}
	go updatePlayer(&player)
}

// 增加金幣
func AddGold(player *redisPlayer, addGold int) {
	player.goldChan <- addGold
}

// 增加金幣
func AddExp(player *redisPlayer, addExp int) {
	player.expChan <- addExp
}

// 暫存資料寫入並每X毫秒更新上RedisDB
func updatePlayer(player *redisPlayer) {
	ticker := time.NewTicker(time.Duration(dbWriteMinMiliSecs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case goldChange := <-player.goldChan:
			player.goldBalance += goldChange
		case expChange := <-player.expChan:
			player.expBalance += expChange
		case <-ticker.C:
			player.writePlayerUpdateToRedis()
		case <-ctx.Done():
			return
		}
	}
}

// 顯示Redis Player資料
func ShowPlayer(playerID string) {

	val, err := rdb.HGetAll(ctx, playerID).Result()
	if err != nil {
		log.Errorf("ShowPlayer錯誤: %v", err)
	}
	id := val["id"]
	gold, _ := strconv.ParseInt(val["gold"], 10, 64)
	heroExp, _ := strconv.ParseInt(val["heroExp"], 10, 64)
	log.Infof("%s playerID: %s gold: %d heroExp: %d\n", logger.LOG_Redis, id, gold, heroExp)
}
