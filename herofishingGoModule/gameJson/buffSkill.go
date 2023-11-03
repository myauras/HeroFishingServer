package gameJson

import (
	"encoding/json"
	"fmt"
)

// BuffSkill JSON
type BuffSkillJsonData struct {
	ID  string  `json:"ID"`
	RTP string `json:"RTP"`
	// Motion string `json:"Motion,omitempty"`
}

func (jsonData BuffSkillJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]BuffSkillJsonData
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

func GetBuffSkills() ([]BuffSkillJsonData, error) {
	datas, err := getJsonDataByName(JsonName.BuffSkill) // Assuming you have JsonName.BuffSkill defined
	if err != nil {
		return nil, err
	}

	var buffSkills []BuffSkillJsonData
	for _, data := range datas {
		if buffSkill, ok := data.(BuffSkillJsonData); ok {
			buffSkills = append(buffSkills, buffSkill)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return buffSkills, nil
}

func GetBuffSkillByID(id string) (BuffSkillJsonData, error) {
	buffSkills, err := GetBuffSkills()
	if err != nil {
		return BuffSkillJsonData{}, err
	}

	for _, buffSkill := range buffSkills {
		if buffSkill.ID == id {
			return buffSkill, nil
		}
	}

	return BuffSkillJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.BuffSkill)
}
