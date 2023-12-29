package gameJson

import (
	"encoding/json"
	"fmt"
)

// DropSpell JSON
type DropSpellJsonData struct {
	ID           string `json:"ID"`
	RTP          string `json:"RTP"`
	EffectType   string `json:"EffectType"`
	EffectValue1 string `json:"EffectValue1,omitempty"`
	EffectValue2 string `json:"EffectValue2,omitempty"`
}

func (jsonData DropSpellJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]DropSpellJsonData
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

func GetDropSpells() ([]DropSpellJsonData, error) {
	datas, err := getJsonDataByName(JsonName.DropSpell)
	if err != nil {
		return nil, err
	}

	var buffSkills []DropSpellJsonData
	for _, data := range datas {
		if buffSkill, ok := data.(DropSpellJsonData); ok {
			buffSkills = append(buffSkills, buffSkill)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return buffSkills, nil
}

func GetDropSpellByID(id string) (DropSpellJsonData, error) {
	buffSkills, err := GetDropSpells()
	if err != nil {
		return DropSpellJsonData{}, err
	}

	for _, buffSkill := range buffSkills {
		if buffSkill.ID == id {
			return buffSkill, nil
		}
	}

	return DropSpellJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.DropSpell)
}
