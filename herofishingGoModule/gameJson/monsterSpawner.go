package gameJson

import (
	"encoding/json"
	"fmt"
	"herofishingGoModule/logger"
	"herofishingGoModule/utility"

	"github.com/google/martian/log"
)

// MonsterSpawner JSON
type MonsterSpawnerJsonData struct {
	ID                      string `json:"ID"`
	SpawnType               string `json:"SpawnType"`
	TypeValue               string `json:"TypeValue"`
	MonsterIDs              string `json:"MonsterIDs"`
	MonsterSpawnIntervalSec string `json:"MonsterSpawnIntervalSec"`
	Routes                  string `json:"Routes"`
}

func (jsonData MonsterSpawnerJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]MonsterSpawnerJsonData
	if err := json.Unmarshal(jsonBytes, &wrapper); err != nil {
		return nil, err
	}

	datas, ok := wrapper[jsonName]
	if !ok {
		return nil, fmt.Errorf("找不到key值: %s", jsonName)
	}

	items := make(map[string]interface{})
	for _, item := range datas {
		items[item.ID] = item
	}
	return items, nil
}

func GetMonsterSpawners() ([]MonsterSpawnerJsonData, error) {
	datas, err := getJsonDataByName(JsonName.MonsterSpawner)
	if err != nil {
		return nil, err
	}

	var monsterSpawners []MonsterSpawnerJsonData
	for _, data := range datas {
		if myMonsterSpawner, ok := data.(MonsterSpawnerJsonData); ok {
			monsterSpawners = append(monsterSpawners, myMonsterSpawner)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return monsterSpawners, nil
}

func GetMonsterSpawnerByID(id string) (MonsterSpawnerJsonData, error) {
	monsterSpawners, err := GetMonsterSpawners()
	if err != nil {
		return MonsterSpawnerJsonData{}, err
	}

	for _, myMonsterSpawner := range monsterSpawners {
		if myMonsterSpawner.ID == id {
			return myMonsterSpawner, nil
		}
	}

	return MonsterSpawnerJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.MonsterSpawner)
}

// 取得隨機生怪秒數
func (jsonData MonsterSpawnerJsonData) GetRandSpawnSec() (float64, error) {
	ids, err := utility.StrToIntSlice(jsonData.MonsterSpawnIntervalSec)
	if len(ids) != 2 {
		return 0, err
	}
	rand, err := utility.RandomFloatBetweenInts(ids[0], ids[1])
	if err != nil {
		log.Errorf("%s GetRandSpawnSec錯誤: %v", logger.LOG_GameJson)
	}
	return rand, nil
}
