package main

// import (
// 	"context"
// 	"herofishingGoModule/matchgrpc"
// 	"io"
// 	logger "matchmaker/logger"

// 	log "github.com/sirupsen/logrus"
// 	"google.golang.org/grpc"
// )

// type matchServer struct {
// 	matchgrpc.UnimplementedMatchServiceServer
// }

// type GRPCConn struct {
// 	Stream matchgrpc.MatchService_MatchCommClient
// 	Conn   *grpc.ClientConn
// }

// var GRPCConns map[string]GRPCConn

// func (s *matchServer) BiDirectionalStream(stream matchgrpc.MatchService_MatchCommServer) error {
// 	for {
// 		in, err := stream.Recv()
// 		if err == io.EOF {
// 			return nil
// 		}
// 		if err != nil {
// 			log.Errorf("%s 接收訊息錯誤: %v", logger.LOG_gRPC, err)
// 			return nil
// 		}

// 		log.Infof("%s 接收訊息: %s", logger.LOG_gRPC, in.GetMessage())

// 		// 發送
// 		if err := stream.Send(&matchgrpc.MatchResponse{Message: "Response from MatchMaker"}); err != nil {
// 			log.Errorf("%s 送訊息錯誤: %v", logger.LOG_gRPC, err)
// 		}
// 	}
// }

// func NewGRPCConnection(address string) {
// 	log.Infof("%s 建立gRPC連線: %s", logger.LOG_gRPC, address)
// 	// 檢查 GRPCConns 是否已經初始化
// 	if GRPCConns == nil {
// 		GRPCConns = make(map[string]GRPCConn)
// 	}

// 	// 建立到指定 matchgame 服務的 gRPC 連線
// 	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
// 	if err != nil {
// 		log.Errorf("%s 無法連接到 %s: %v", logger.LOG_gRPC, address, err)
// 		return
// 	}

// 	// 創建 MatchService 客戶端
// 	client := matchgrpc.NewMatchServiceClient(conn)

// 	// 建立雙向流
// 	stream, err := client.MatchComm(context.Background())
// 	if err != nil {
// 		log.Errorf("%s 無法建立與 %s 的雙向流: %v", logger.LOG_gRPC, address, err)
// 	}

// 	GRPCConns[address] = GRPCConn{
// 		Stream: stream,
// 		Conn:   conn,
// 	}

// 	// 啟動一個 goroutine 來接收來自該連線的訊息
// 	go Receive(stream, address)
// }

// func Receive(str matchgrpc.MatchService_MatchCommClient, addr string) {
// 	for {
// 		in, err := str.Recv()
// 		if err == io.EOF {
// 			log.Infof("%s 與 %s 的連線被關閉", logger.LOG_gRPC, addr)
// 			return
// 		}
// 		if err != nil {
// 			log.Errorf("%s 從 %s 接收訊息錯誤: %v", logger.LOG_gRPC, addr, err)
// 			return
// 		}

// 		log.Infof("%s 從 %s 收到訊息: %s", logger.LOG_gRPC, addr, in.GetMessage())
// 	}
// }

// func Send(message string, address string) {
// 	grpcConn, ok := GRPCConns[address]
// 	if !ok {
// 		log.Errorf("%s 未找到 %s 的連接", logger.LOG_gRPC, address)
// 		return
// 	}

// 	if err := grpcConn.Stream.Send(&matchgrpc.MatchRequest{Message: message}); err != nil {
// 		log.Errorf("%s 向 %s 發送錯誤: %v", logger.LOG_gRPC, address, err)
// 	}
// }

// func CloseGRPCConnection(address string) {
// 	if GRPCConns == nil {
// 		log.Errorf("%s GRPC 連線尚未初始化", logger.LOG_gRPC)
// 		return
// 	}

// 	// 查找並關閉特定地址的連線
// 	if connData, exists := GRPCConns[address]; exists {
// 		if connData.Conn != nil {
// 			// 關閉 gRPC 連線
// 			err := connData.Conn.Close()
// 			if err != nil {
// 				log.Errorf("%s 關閉 %s 的 gRPC 連線時出錯: %v", logger.LOG_gRPC, address, err)
// 			}
// 		}

// 		// 從字典中移除該條目
// 		delete(GRPCConns, address)
// 	} else {
// 		log.Infof("%s 沒有找到與 %s 相關的 gRPC 連線", logger.LOG_gRPC, address)
// 	}
// }
