package redis

import (
	// "context"
	"fmt"
	logger "herofishingGoModule/logger"
	// redis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

// 訂閱Redis訊息
func Subscribe(channelName string, subscribeMsgChan chan interface{}) error {
	pubsub = rdb.Subscribe(ctx, channelName)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("%s 訂閱 %s通道 失敗: %s", logger.LOG_Redis, channelName, err)
	}
	go receiveSubscribeMsg(channelName, subscribeMsgChan)
	return nil
}

func receiveSubscribeMsg(channelName string, subscribeMsgChan chan interface{}) {
	for msg := range pubsub.Channel() {
        log.Infof("%s 收到Redis %s通道 訊息: %s", logger.LOG_Redis, channelName, msg.Payload)
        subscribeMsgChan <- msg.Payload // 发送消息内容而非频道名
    }
}


// 推送Redis訊息
func Publish(channelName string, msg interface{}) error {
	err := rdb.Publish(ctx, channelName, msg).Err()
	if err != nil {
		return fmt.Errorf("%s 推送Redis %s通道 訊息: %s", logger.LOG_Redis, channelName, msg)
	}
	return nil
}
