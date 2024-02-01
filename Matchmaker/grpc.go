package main

import (
	"herofishingGoModule/matchgrpc"
	"io"
	logger "matchmaker/logger"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type matchServer struct {
	matchgrpc.UnimplementedMatchServiceServer
}

func (s *matchServer) BiDirectionalStream(stream matchgrpc.MatchService_MatchCommServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Errorf("%s 接收訊息錯誤: %v", logger.LOG_gRPC, err)
		}

		log.Infof("%s 接收訊息: %s", logger.LOG_gRPC, in.GetMessage())

		// 发送响应
		if err := stream.Send(&matchgrpc.MatchResponse{Message: "Response from MatchMaker"}); err != nil {
			log.Errorf("%s 送訊息錯誤: %v", logger.LOG_gRPC, err)
		}
	}
}

func NewGRPCConnn() {
	listen, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Errorf("%s 偵聽錯誤: %v", logger.LOG_gRPC, err)
	}
	s := grpc.NewServer()
	matchgrpc.RegisterMatchServiceServer(s, &matchServer{})
	log.Infof("%s 開始偵聽 %v", logger.LOG_gRPC, listen.Addr())
	if err := s.Serve(listen); err != nil {
		log.Errorf("%s Serve錯誤 %v", logger.LOG_gRPC, err)
	}
}
