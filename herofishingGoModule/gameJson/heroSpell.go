package gameJson

import (
	"encoding/json"
	"fmt"
	// "herofishingGoModule/logger"
)

// HeroSpell JSON
type HeroSpellJsonData struct {
	ID    string `json:"ID"`
	RTP   string `json:"RTP"`
	CD    string `json:"CD"`
	Cost  string `json:"Cost"`
	Waves string `json:"Waves"`
	Hits  string `json:"Hits"`
}

func (jsonData HeroSpellJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]HeroSpellJsonData
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

func GetHeroSpells() ([]HeroSpellJsonData, error) {
	datas, err := getJsonDataByName(JsonName.HeroSpell)
	if err != nil {
		return nil, err
	}

	var heroSpells []HeroSpellJsonData
	for _, data := range datas {
		if hero, ok := data.(HeroSpellJsonData); ok {
			heroSpells = append(heroSpells, hero)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return heroSpells, nil
}

func GetHeroSpellByID(id string) (HeroSpellJsonData, error) {
	heroSpells, err := GetHeroSpells()
	if err != nil {
		return HeroSpellJsonData{}, err
	}

	for _, heroSpell := range heroSpells {
		if heroSpell.ID == id {
			return heroSpell, nil
		}
	}

	return HeroSpellJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.HeroSpell)
}
