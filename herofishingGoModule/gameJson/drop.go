package gameJson

import (
	"encoding/json"
	"fmt"
)

// Drop JSON
type DropJsonData struct {
	ID string `json:"ID"`
	// Ref          string `json:"Ref"`
	RTP       string `json:"RTP"`
	DropType  string `json:"DropType"`
	DropValue string `json:"DropValue,omitempty"`
}

func (jsonData DropJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]DropJsonData
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

func GetDrops() ([]DropJsonData, error) {
	datas, err := getJsonDataByName(JsonName.Drop)
	if err != nil {
		return nil, err
	}

	var drops []DropJsonData
	for _, data := range datas {
		if drop, ok := data.(DropJsonData); ok {
			drops = append(drops, drop)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return drops, nil
}

func GetDropByID(id string) (DropJsonData, error) {
	drops, err := GetDrops()
	if err != nil {
		return DropJsonData{}, err
	}

	for _, drop := range drops {
		if drop.ID == id {
			return drop, nil
		}
	}

	return DropJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.Drop)
}
