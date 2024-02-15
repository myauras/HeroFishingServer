package gameJson

import (
	"encoding/json"
	"fmt"
	"strconv"
	// "herofishingGoModule/logger"
)

// HeroEXP JSON
type HeroEXPJsonData struct {
	ID  string `json:"ID"`
	EXP string `json:"EXP"`
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
		if heroEXP, ok := data.(HeroEXPJsonData); ok {
			heroEXPs = append(heroEXPs, heroEXP)
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

// 傳入經驗取得等級
func GetHeroLVByEXP(exp int32) (int32, error) {
	datas, err := getJsonDataByName(JsonName.HeroEXP)
	if err != nil {
		return 1, err
	}

	privousLV := int32(1)
	for _, data := range datas {
		if expJson, ok := data.(HeroEXPJsonData); ok {
			lv, lvErr := strconv.ParseInt(expJson.ID, 10, 32)
			if lvErr != nil {
				return 1, fmt.Errorf("strconv.ParseInt(exp.ID, 10, 32)錯誤: %s", lvErr)
			}
			needEXP, expErr := strconv.ParseInt(expJson.EXP, 10, 32)
			if expErr != nil {
				return 1, fmt.Errorf("strconv.ParseInt(exp.ID, 10, 32)錯誤: %s", expErr)
			}
			if int64(exp) < needEXP {
				return privousLV, nil
			}
			privousLV = int32(lv)
		} else {
			return 1, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return privousLV, nil
}
