package gameJson

import (
	"encoding/json"
	"fmt"
	// "herofishingGoModule/logger"
)

// HeroEXP JSON
type HeroEXPJsonData struct {
	ID  string `json:"ID"`
	EXP string    `json:"EXP"`
}

func (jsonData HeroEXPJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]HeroEXPJsonData
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

func GetHeroEXPs() ([]HeroEXPJsonData, error) {
	datas, err := getJsonDataByName(JsonName.HeroEXP)
	if err != nil {
		return nil, err
	}

	var heroEXPs []HeroEXPJsonData
	for _, data := range datas {
		if hero, ok := data.(HeroEXPJsonData); ok {
			heroEXPs = append(heroEXPs, hero)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return heroEXPs, nil
}

func GetHeroEXPByID(id string) (HeroEXPJsonData, error) {
	heroEXPs, err := GetHeroEXPs()
	if err != nil {
		return HeroEXPJsonData{}, err
	}

	for _, heroEXP := range heroEXPs {
		if heroEXP.ID == id {
			return heroEXP, nil
		}
	}

	return HeroEXPJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.HeroEXP)
}
