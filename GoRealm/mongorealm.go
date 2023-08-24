package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.TODO()
	uri := "mongodb+srv://cluster0.edk0n6b.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=../Untracked/X509-cert-6644120643259731564.pem"
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(serverAPIOptions)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	collection := client.Database("herofishing").Collection("player")
	docCount, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(docCount)
}
