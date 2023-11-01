package gameJson

import (
	"context"
	"fmt"
	"herofishingGoModule/logger"
	"herofishingGoModule/setting"
	"io/ioutil"
	"strings"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

// 初始化JsonMap
func Init(env string) error {
	gcpProjectID, ok := setting.EnvGCPProject[env]
	if !ok {
		// log.Errorf("%s env錯誤: %s", logger.LOG_GameJson, env)
		return fmt.Errorf("%s evn名稱錯誤: %v", logger.LOG_GameJson, env)
	}
	log.Infof("%s gcpProjectID: %s", logger.LOG_GameJson, gcpProjectID)

	ctx := context.Background()

	// 初始化 GCS 客戶端
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("%s GCS初始化錯誤: %v", logger.LOG_GameJson, err)
	}

	// 設定bucket和object前綴
	bucketName := "herofishing_gamejson_dev"
	prefix := "" // 如果所有的json都在根目錄，就用空字串就可以

	bucket := client.Bucket(bucketName)
	// 創建一個列舉object的查詢
	query := &storage.Query{Prefix: prefix}

	// 執行查詢
	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("%s ListBucket: %v", logger.LOG_GameJson, err)
		}

		// 檢查是否為.json檔案
		if len(attrs.Name) > 5 && attrs.Name[len(attrs.Name)-5:] == ".json" {
			object := bucket.Object(attrs.Name)
			reader, err := object.NewReader(ctx)
			if err != nil {
				return fmt.Errorf("%s Failed to read object: %v", logger.LOG_GameJson, err)
			}
			defer reader.Close()

			data, err := ioutil.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("%s Failed to read data: %v", logger.LOG_GameJson, err)
			}
			jsonName := strings.TrimSuffix(attrs.Name, ".json")
			// fmt.Printf("%s File: %s Data: %s \n", logger.LOG_GameJson, attrs.Name, data)
			SetJsonDic(jsonName, data)

		}
	}

	return nil
}
