package mongo

import (
	"context"
	"fmt"
	logger "herofishingGoModule/logger"
	"time"

	log "github.com/sirupsen/logrus"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	DB     *mongoDriver.Database
	Client *mongoDriver.Client
)

type InitData struct {
	Env           string
	APIPublicKey  string
	APIPrivateKey string
}

func Init(data InitData, user string, pw string) {
	Env = data.Env
	APIPublicKey = data.APIPublicKey
	APIPrivateKey = data.APIPrivateKey
	connToMongoDB(user, pw) // 連線MongoDB
}

// 連線MongoDB
func connToMongoDB(user string, pw string) error {
	if Client != nil {
		return fmt.Errorf("MongoDB client已經被建立, 不可重複建立")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	serverAPIOpt := options.ServerAPI(options.ServerAPIVersion1)
	uri := fmt.Sprintf(EnvDBUri[Env], user, pw)
	clientOpt := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPIOpt)
	Client, err := mongoDriver.Connect(ctx, clientOpt)
	if err != nil {
		log.Errorf("%s 連線Mongo DB失敗: %v", logger.LOG_Mongo, err)
		return err
	}
	DB = Client.Database(EnvDB[Env])
	return nil
}

// 斷開與MongoDB的連線
func DisconnectFromMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if Client != nil {
		Client.Disconnect(ctx)
	}
}
