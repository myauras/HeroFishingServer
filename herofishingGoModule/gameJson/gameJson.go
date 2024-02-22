package gameJson

import (
	"context"
	"herofishingGoModule/setting"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"

	"errors"
	"fmt"
	"herofishingGoModule/logger"
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
	bucketName := "herofishing_gamejson_dev3"
	prefix := "" // 如果所有的json都在根目錄，就用空字串就可以

	bucket := client.Bucket(bucketName)
	// 創建一個列舉object的查詢
	query := &storage.Query{Prefix: prefix}

	// 執行查詢
	item := bucket.Objects(ctx, query)
	index := 0
	for {
		attrs, err := item.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("%s ListBucket: %v", logger.LOG_GameJson, err)
		}

		// 檢查是否為.json檔案
		if len(attrs.Name) > 5 && attrs.Name[len(attrs.Name)-5:] == ".json" {
			index++
			object := bucket.Object(attrs.Name)
			reader, err := object.NewReader(ctx)
			if err != nil {
				return fmt.Errorf("%s Failed to read object: %v", logger.LOG_GameJson, err)
			}

			data, err := io.ReadAll(reader)
			reader.Close()
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

// jsonDic的結構為jsonDic[jsonName][ID]
var jsonDic = make(map[string]map[string]interface{})

type JsonNameStruct struct {
	GameSetting    string
	Hero           string
	HeroEXP        string
	HeroSpell      string
	Map            string
	Monster        string
	MonsterSpawner string
	Route          string
	DropSpell      string
	Drop           string
	Rank           string
}

// Json名稱列表
var JsonName = JsonNameStruct{
	GameSetting:    "GameSetting",
	Hero:           "Hero",
	HeroEXP:        "HeroEXP",
	HeroSpell:      "HeroSpell",
	Map:            "Map",
	Monster:        "Monster",
	MonsterSpawner: "MonsterSpawner",
	Route:          "Route",
	DropSpell:      "DropSpell",
	Drop:           "Drop",
	Rank:           "Rank",
}

// 傳入Json名稱取得對應JsonMap資料
func getJsonDataByName(name string) (map[string]interface{}, error) {
	data, exists := jsonDic[name]
	if !exists {
		return nil, fmt.Errorf("jsonDic中未找到 %s 的資料", name)
	}
	return data, nil
}

type JsonUnmarshaler interface {
	UnmarshalJSONData(jsonName string, sonData []byte) (map[string]interface{}, error)
}

// 傳入Json將並轉為對應struct資料並存入jsonDic中, jsonDic的結構為jsonDic[jsonName][ID]
func SetJsonDic(jsonName string, jsonData []byte) error {

	var unmarshaler JsonUnmarshaler
	switch jsonName {
	case JsonName.GameSetting:
		unmarshaler = GameSettingJsonData{}
	case JsonName.Hero:
		unmarshaler = HeroJsonData{}
	case JsonName.HeroSpell:
		unmarshaler = HeroSpellJsonData{}
	case JsonName.HeroEXP:
		unmarshaler = HeroEXPJsonData{}
	case JsonName.Map:
		unmarshaler = MapJsonData{}
	case JsonName.Monster:
		unmarshaler = MonsterJsonData{}
	case JsonName.MonsterSpawner:
		unmarshaler = MonsterSpawnerJsonData{}
	case JsonName.Route:
		unmarshaler = RouteJsonData{}
	case JsonName.DropSpell:
		unmarshaler = DropSpellJsonData{}
	case JsonName.Drop:
		unmarshaler = DropJsonData{}
	case JsonName.Rank:
		unmarshaler = RankJsonData{}
	default:
		log.Errorf("%s 未定義的jsonName: %v", logger.LOG_GameJson, jsonName)
		return errors.New("未定義的jsonName")
	}
	items, err := unmarshaler.UnmarshalJSONData(jsonName, jsonData)
	if err != nil {
		log.Errorf("%s %s表Unmarshal失敗: %v", logger.LOG_GameJson, jsonName, err)
		return err
	}
	jsonDic[jsonName] = items
	log.Infof("%s 設定Json資料: %s", logger.LOG_GameJson, jsonName)
	return nil
}
