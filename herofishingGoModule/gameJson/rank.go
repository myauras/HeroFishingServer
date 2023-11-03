package gameJson

import (
	"encoding/json"
	"fmt"
)

// / Rank JSON
type RankJsonData struct {
	ID    string `json:"ID"`
	Point string    `json:"Point"`
}

func (jsonData RankJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]RankJsonData
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

func GetRanks() ([]RankJsonData, error) {
	datas, err := getJsonDataByName(JsonName.Rank) // Assuming you have JsonName.Rank defined
	if err != nil {
		return nil, err
	}

	var ranks []RankJsonData
	for _, data := range datas {
		if rank, ok := data.(RankJsonData); ok {
			ranks = append(ranks, rank)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return ranks, nil
}

func GetRankByID(id string) (RankJsonData, error) {
	ranks, err := GetRanks()
	if err != nil {
		return RankJsonData{}, err
	}

	for _, rank := range ranks {
		if rank.ID == id {
			return rank, nil
		}
	}

	return RankJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.Rank)
}
