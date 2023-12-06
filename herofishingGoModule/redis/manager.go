package redis

import (
	"context"
	"fmt"
	logger "herofishingGoModule/logger"

	redis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

var rdb *redis.Client
var ctx context.Context
var cancel context.CancelFunc
var pubsub *redis.PubSub

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

// 訂閱Redis訊息
func Subscribe(channelName string, subscribeMsgChan chan string) error {
	pubsub = rdb.Subscribe(ctx, channelName)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("%s 訂閱 %s通道 失敗: %s", logger.LOG_Redis, channelName, err)
	}
	go receiveSubscribeMsg(channelName, subscribeMsgChan)
	return nil
}

func receiveSubscribeMsg(channelName string, subscribeMsgChan chan string) {
	for msg := range pubsub.Channel() {
		log.Infof("%s 收到Redis %s通道 訊息: %s", logger.LOG_Redis, channelName, msg)
		subscribeMsgChan <- msg.Channel
	}
}

// 推送Redis訊息
func Publish(channelName string, msg string) error {
	err := rdb.Publish(ctx, channelName, msg).Err()
	if err != nil {
		return fmt.Errorf("%s 推送Redis %s通道 訊息: %s", logger.LOG_Redis, channelName, msg)
	}
	return nil
}
