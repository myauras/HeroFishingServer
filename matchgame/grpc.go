package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"herofishingGoModule/matchgrpc"
	logger "matchgame/logger"
	"time"
)

func NewGRPCConnn() {
	conn, err := grpc.Dial(":50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Errorf("%s 連接失敗: %v", logger.LOG_gRPC, err)
	}
	defer conn.Close()
	client := matchgrpc.NewMatchServiceClient(conn)

	stream, err := client.MatchComm(context.Background())
	if err != nil {
		log.Errorf("%s 建立雙向串流: %v", logger.LOG_gRPC, err)
	}

	// 發送
	go func() {
		for {
			if err := stream.Send(&matchgrpc.MatchRequest{Sender: "matchgame", Message: "Hello from MatchGame"}); err != nil {
				log.Errorf("%s 發送訊息失敗: %v", logger.LOG_gRPC, err)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// 接收
	for {
		response, err := stream.Recv()
		if err != nil {
			log.Errorf("%s 接收訊息失敗: %v", logger.LOG_gRPC, err)
			break
		}
		log.Infof("%s 收到訊息: %v", logger.LOG_gRPC, response.GetMessage())
	}
}
