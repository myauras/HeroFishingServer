package redis

import (
	"context"
	// "fmt"
	redis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	logger "herofishingGoModule/logger"
	"time"
)

var rdb *redis.Client
var ctx context.Context
var Cancel context.CancelFunc
var pubsub *redis.PubSub

// 初始化RedisDB
func Init() {
	log.Infof("%s 開始初始化RedisDB", logger.LOG_Redis)
	if rdb != nil {
		return
	}
	newRedisClient()
	ctx, Cancel = context.WithCancel(context.Background())
	players = make(map[string]*RedisPlayer)
	redisErr := Ping()
	if redisErr != nil {
		log.Errorf("%s 初始化RedisDB發生錯誤: %v", logger.LOG_Redis, redisErr)
	} else {
		log.Infof("%s 初始化RedisDB完成", logger.LOG_Redis)
	}
}
func newRedisClient() {
	rdb = redis.NewClient(&redis.Options{
		Addr:            "redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com:10238",
		Password:        "dMfmpIDd0BTIyeCnOkBhuznVPxd7V7yx",
		DB:              0,
		ConnMaxIdleTime: 3 * time.Minute, // 閒置斷線時間, 超過這個時間不使用就自動與DB斷連
	})
}

func Ping() error {
	var err error
	for i := 0; i < 5; i++ {
		_, err = rdb.Ping(ctx).Result()
		if err == nil {
			return nil
		} else {
			log.Warnf("%s Ping RedisDB失敗: %v", logger.LOG_Redis, err)
		}
		time.Sleep(1 * time.Second)
	}
	return err
}
