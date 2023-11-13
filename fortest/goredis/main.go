package main

import (
	"context"
	"time"
	// "crypto/tls"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

// Test-NetConnection -ComputerName redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com -Port 10238

func main() {

	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-10238.c302.asia-northeast1-1.gce.cloud.redislabs.com:10238",
		Password: "dMfmpIDd0BTIyeCnOkBhuznVPxd7V7yx", // no password set
		DB:       0,                                  // use default DB
		// TLSConfig: &tls.Config{
		// 	MinVersion: tls.VersionTLS12,
		// 	//Certificates: []tls.Certificate{cert}
		// },
	})

	ctx := context.Background()
	rdb.Set(ctx, "name", "scoz2", 1*time.Minute)
	val, err := rdb.Do(ctx, "get", "name").Result()
	if err != nil {
		if err == redis.Nil {
			fmt.Println("key does not exists")
			return
		}
		panic(err)
	}
	fmt.Println(val.(string))

	// val, err := rdb.Get(ctx, "key").Result()
	// switch {
	// case err == redis.Nil:
	// 	fmt.Println("key does not exist")
	// case err != nil:
	// 	fmt.Println("Get failed", err)
	// case val == "":
	// 	fmt.Println("value is empty")
	// }

	// Alternatively you can save the command and later access the value and the error separately:
	// get := rdb.Get(ctx, "key")
	// fmt.Println(get.Val(), get.Err())
}
