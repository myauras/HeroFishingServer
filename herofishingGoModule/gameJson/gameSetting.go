package gameJson

import (
	"encoding/json"
	"fmt"
	// "herofishingGoModule/logger"
)

// GameSetting JSON
type GameSettingJsonData struct {
	ID    string `json:"ID"`
	Value string `json:"Value"`
}

func (jsonData GameSettingJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]GameSettingJsonData
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
