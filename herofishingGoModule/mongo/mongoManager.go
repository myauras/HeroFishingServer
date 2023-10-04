package mongo

type InitData struct {
	Env           string
	APIPublicKey  string
	APIPrivateKey string
}

func Init(data InitData) {
	Env = data.Env
	APIPublicKey = data.APIPublicKey
	APIPrivateKey = data.APIPrivateKey
}

// import (
// 	"context"
// 	"fmt"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"
// 	"log"
// 	"time"
// )

// func Init() {

// 	// 連線MongoDB
// 	connStr := "mongodb+srv://cluster0.edk0n6b.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority"
// 	clientOpt := options.Client().ApplyURI(connStr)
// 	// 建立MongoDB client
// 	client, err := mongo.NewClient(clientOpt)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// 設定context以及連接到MongoDB
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	err = client.Connect(ctx)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer client.Disconnect(ctx)

// 	// 呼叫Atlas function
// 	// 假設有一個叫做 "myFunction" 的function
// 	database := client.Database("your_database_name")
// 	result, err := database.RunCommand(ctx, bson.D{{"eval", "myFunction()"}}).DecodeBytes()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println(result.String())
// }
