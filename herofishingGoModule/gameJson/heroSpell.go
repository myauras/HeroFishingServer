package gameJson

import (
	"encoding/json"
	"fmt"
	"strconv"
	// "herofishingGoModule/logger"
)

// HeroSpell JSON
type HeroSpellJsonData struct {
	ID      string  `json:"ID"`
	RTP     float64 `json:"RTP"`
	CD      float64 `json:"CD"`
	Cost    int32   `json:"Cost"`
	MaxHits int32   `json:"MaxHits"`
}

func (jsonData HeroSpellJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &wrapper); err != nil {
		return nil, err
	}

	rawDatas, ok := wrapper[jsonName]
	if !ok {
		return nil, fmt.Errorf("找不到key值: %s", jsonName)
	}

	items := make(map[string]interface{})
	for _, rawData := range rawDatas {
		var item HeroSpellJsonData
		if err := json.Unmarshal(rawData, &item); err != nil {
			return nil, err
		}
		items[item.ID] = item
	}
	return items, nil
}

func (spellJson *HeroSpellJsonData) UnmarshalJSON(data []byte) error {
	type Alias HeroSpellJsonData
	aux := &struct {
		RTP     string `json:"RTP"`
		CD      string `json:"CD"`
		Cost    string `json:"Cost"`
		MaxHits string `json:"MaxHits"`
		*Alias
	}{
		Alias: (*Alias)(spellJson), // 使用Alias避免在UnmarshalJSON中呼叫json.Unmarshal時的無限遞迴
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if aux.RTP != "" {
		if spellJson.RTP, err = strconv.ParseFloat(aux.RTP, 64); err != nil {
			return err
		}
	}
	if aux.CD != "" {
		if spellJson.CD, err = strconv.ParseFloat(aux.CD, 64); err != nil {
			return err
		}
	}
	if aux.Cost != "" {
		var intVal int64
		if intVal, err = strconv.ParseInt(aux.Cost, 10, 32); err != nil {
			return err
		}
		spellJson.Cost = int32(intVal)
	}
	if aux.MaxHits != "" {
		var intVal int64
		if intVal, err = strconv.ParseInt(aux.MaxHits, 10, 32); err != nil {
			return err
		}
		spellJson.MaxHits = int32(intVal)
	}

	return nil
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
