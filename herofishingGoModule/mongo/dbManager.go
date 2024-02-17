package mongo

// MongoDB go CRUD參考官方文件: https://www.mongodb.com/docs/drivers/go/current/fundamentals/crud/write-operations/modify/

import (
	"context"
	logger "herofishingGoModule/logger"

	"github.com/google/martian/log"
	"go.mongodb.org/mongo-driver/bson"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 使用Collection與ID來取文件
func GetDocByID(col string, id string, result interface{}) error {

	filter := bson.D{{Key: "_id", Value: id}}

	err := DB.Collection(col).FindOne(context.TODO(), filter).Decode(result)
	if err != nil {
		log.Infof("%s GetDocByID錯誤: %v", logger.LOG_Mongo, err)
		return err
	}
	return nil
}

// GetDocsByFieldValue 取得指定欄位符合特定條件下的所有文件
// fieldValue必須符合fieldName的類型
// results 必須是DB Sechma Struct切片 Ex. []DBPlayer
func GetDocsByFieldValue(col string, fieldName string, fieldValue interface{}, operator Operator, results interface{}) error {

	// 創建查詢過濾器
	filter := bson.M{fieldName: bson.M{string(operator): fieldValue}}

	// 執行查詢
	cursor, err := DB.Collection(col).Find(context.TODO(), filter)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	// 解碼查詢結果到 results 指向的切片
	if err = cursor.All(context.TODO(), results); err != nil {
		return err
	}

	return nil
}

// GetDocsByFieldValue 取得指定欄位符合特定條件下的所有文件的ID切片
// fieldValue必須符合fieldName的類型
func GetDocIDsByFieldValue(col string, fieldName string, fieldValue interface{}, operator Operator) ([]string, error) {
	var results []string

	// 創建查詢過濾器
	filter := bson.M{fieldName: bson.M{string(operator): fieldValue}}
	// 只包括 _id 字段的投影
	projection := bson.M{"_id": 1} // 1是包含, 0是排除

	// 執行查詢只返回 _id
	cursor, err := DB.Collection(col).Find(context.TODO(), filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var elem struct {
			ID string `bson:"_id"`
		}
		if err = cursor.Decode(&elem); err != nil {
			return nil, err
		}
		results = append(results, elem.ID)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetDocIDsByFilter 取得符合特定過濾條件的所有文檔的ID切片
// col: 集合名稱
// filter: 過濾條件
func GetDocIDsByFilter(col string, filter bson.M) ([]string, error) {
	var results []string

	// 只包括 _id 欄位的投影
	projection := bson.M{"_id": 1} // 1是包含, 0是排除

	// 執行查詢只返回符合投影條件的字段
	cursor, err := DB.Collection(col).Find(context.TODO(), filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var elem struct {
			ID string `bson:"_id"`
		}
		if err = cursor.Decode(&elem); err != nil {
			return nil, err
		}
		results = append(results, elem.ID)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// 更新文件
func UpdateDocByBsonD(col string, id string, updateData bson.D) (*mongoDriver.UpdateResult, error) {

	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: updateData}}

	result, err := DB.Collection(col).UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// 更新文件
func UpdateDocByInterface(col string, id string, updateData interface{}) (*mongoDriver.UpdateResult, error) {

	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: updateData}}

	result, err := DB.Collection(col).UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateDocsByField 批量更新指定欄位符合特定條件的所有文檔
// col: 要更新的集合名
// filterField: 過濾條件的欄位名
// filterValue: 過濾條件的欄位值
// updateData: 更新內容 (bson.D 格式) Ex. bson.D{{Key: "onlineState", Value: "Offline"}}
func UpdateDocsByField(col string, filterField string, filterValue interface{}, updateData bson.D) (*mongoDriver.UpdateResult, error) {
	filter := bson.M{filterField: bson.M{"$in": filterValue}}

	result, err := DB.Collection(col).UpdateMany(context.TODO(), filter, bson.D{
		{Key: "$set", Value: updateData},
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// 新增文件
func AddDocByBsonD(col string, addData bson.D) (*mongoDriver.InsertOneResult, error) {

	result, err := DB.Collection(col).InsertOne(context.TODO(), addData)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// 新增文件
func AddDocByStruct(col string, addData interface{}) (*mongoDriver.InsertOneResult, error) {
	result, err := DB.Collection(col).InsertOne(context.TODO(), addData)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// 文件存在就更新不存在就新增
func AddOrUpdateDocByStruct(col string, docID string, addData interface{}) (*mongoDriver.UpdateResult, error) {

	filter := bson.D{{Key: "_id", Value: docID}}
	update := bson.D{{Key: "$set", Value: addData}}

	opts := options.Update().SetUpsert(true)

	result, err := DB.Collection(col).UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		return nil, err
	}
	return result, nil
}
