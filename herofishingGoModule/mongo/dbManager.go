package mongo

// MongoDB go CRUD參考官方文件: https://www.mongodb.com/docs/drivers/go/current/fundamentals/crud/write-operations/modify/

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
)

// 更新文件
func SetDocByID(col string, id string, updateData bson.D) (*mongoDriver.UpdateResult, error) {

	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: updateData}}

	result, err := DB.Collection(col).UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, err
	}

	return result, nil
}
