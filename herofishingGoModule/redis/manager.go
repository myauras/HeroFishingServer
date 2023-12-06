package redis

import (
	"context"
	redis "github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx context.Context
var cancel context.CancelFunc

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
