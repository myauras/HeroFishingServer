package gameJson

import (
	"context"
	"fmt"
	"herofishingGoModule/logger"
	"herofishingGoModule/setting"
	"io/ioutil"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
)

// 初始化JsonMap
func Init(env string) error {

	gcpProjectID, ok := setting.EnvGCPProject[env]
	if !ok {
		log.Errorf("%s env錯誤: %s", logger.LOG_GameJson, env)
		return fmt.Errorf("evn名稱錯誤: %v", env)
	}
	log.Infof("%s gcpProjectID: %s", logger.LOG_GameJson, gcpProjectID)

	ctx := context.Background()

	// 初始化 GCS 客戶端
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("GCS 初始化錯誤: %v", err)
	}

	// 設定你的 bucket 和 object 名稱
	bucketName := "aurafortest_bundle_dev"
	objectName := "Hero.json"

	// 使用 GCS 客戶端從 bucket 下載 object
	bucket := client.Bucket(bucketName)
	object := bucket.Object(objectName)

	reader, err := object.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("GCS reader初始化錯誤: %v", err)
	}
	defer reader.Close()

	// 讀取 object 內容
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		fmt.Printf("Reading object data failed: %v\n", err)
		return fmt.Errorf("GCS 下載檔案失敗: %v", err)
	}

	fmt.Printf("下載檔案成功: %s\n", data)

	return nil
}
