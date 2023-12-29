package gameJson

import (
	"encoding/json"
	"fmt"
	"herofishingGoModule/utility"
	// "herofishingGoModule/logger"
)

// Map JSON
type MapJsonData struct {
	ID                string `json:"ID"`
	Ref               string `json:"Ref"`
	Multiplier        string `json:"Multiplier"`
	MonsterSpawnerIDs string `json:"MonsterSpawnerIDs"`
}

func (jsonData MapJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]MapJsonData
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

func GetMaps() ([]MapJsonData, error) {
	datas, err := getJsonDataByName(JsonName.Map)
	if err != nil {
		return nil, err
	}

	var maps []MapJsonData
	for _, data := range datas {
		if myMap, ok := data.(MapJsonData); ok {
			maps = append(maps, myMap)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return maps, nil
}

func GetMapByID(id string) (MapJsonData, error) {
	maps, err := GetMaps()
	if err != nil {
		return MapJsonData{}, err
	}

	for _, myMap := range maps {
		if myMap.ID == id {
			return myMap, nil
		}
	}

	return MapJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.Map)
}

// 取得此地圖的生怪IDs
func (jsonData MapJsonData) GetMonsterSpawnerIDs() ([]int, error) {
	ids, err := utility.Split_INT(jsonData.MonsterSpawnerIDs, ",")
	return ids, err
}
