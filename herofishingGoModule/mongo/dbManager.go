package mongo

// MongoDB go CRUD參考官方文件: https://www.mongodb.com/docs/drivers/go/current/fundamentals/crud/write-operations/modify/

import (
	"context"
	// "encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
)

// 取文件
func GetDocByID(col string, id string, result interface{}) error {

	filter := bson.D{{Key: "_id", Value: id}}

	err := DB.Collection(col).FindOne(context.TODO(), filter).Decode(result)
	if err != nil {
		return err
	}

	// // 序列化結果為JSON並輸出
	// resultJson, err := json.Marshal(result)
	// if err != nil {
	// 	return "", err
	// }
	// return string(resultJson), nil
	return nil
}

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
