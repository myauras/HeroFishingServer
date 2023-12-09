package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var playerID = "scozirge"
var dbWriteMinMiliSecs = 1000

// Test-NetConnection -ComputerName redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com -Port 10238

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com:10238",
		Password: "dMfmpIDd0BTIyeCnOkBhuznVPxd7V7yx",
		DB:       0,
	})

	ctx, cancel := context.WithCancel(context.Background())

	// _, err := rdb.HMSet(ctx, playerID, map[string]interface{}{
	// 	"gold":   9800,
	// 	"heroLV": 10,
	// }).Result()
	// if err != nil {
	// 	panic(err)
	// }

	defer cancel()

	pointChanges := make(chan int)

	go updatePoint(ctx, rdb, pointChanges)
	// 點數寫入
	pointChanges <- -1
	pointChanges <- 100
	pointChanges <- -1

	time.Sleep(1 * time.Second)
	showPlayerInfo(ctx, rdb) // 顯示最新DB資料

	cancel()
	close(pointChanges)
}

func showPlayerInfo(ctx context.Context, rdb *redis.Client) {

	val, err := rdb.HGetAll(ctx, playerID).Result()
	if err != nil {
		panic(err)
	}

	gold, _ := strconv.ParseInt(val["gold"], 10, 64)
	heroLV64, _ := strconv.ParseInt(val["heroLV"], 10, 32)
	heroLV := int32(heroLV64)
	fmt.Printf("playerID: %s gold: %d heroLV: %d\n", playerID, gold, heroLV)
}

// 暫存點數寫入並每X毫秒更新上RedisDB
func updatePoint(ctx context.Context, rdb *redis.Client, goldChanges <-chan int) {
	var balance int
	ticker := time.NewTicker(time.Duration(dbWriteMinMiliSecs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case change := <-goldChanges:
			balance += change

		case <-ticker.C:
			if balance != 0 {
				_, err := rdb.HIncrBy(ctx, playerID, "gold", int64(balance)).Result()
				if err != nil {
					fmt.Println("Error updating gold:", err)
				}
				balance = 0
			}

		case <-ctx.Done():
			return
		}
	}
}
