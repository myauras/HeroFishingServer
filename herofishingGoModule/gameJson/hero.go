package gameJson

import (
	"encoding/json"
	"fmt"
	"strconv"
	// "herofishingGoModule/logger"
)

// Hero JSON
type HeroJsonData struct {
	ID string `json:"ID"`
	// Ref          string `json:"Ref"`
	// RoleCategory string `json:"RoleCategory"`
	// IdleMotions  string `json:"IdleMotions"`
}

func (jsonData HeroJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]HeroJsonData
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

func GetHeros() ([]HeroJsonData, error) {
	datas, err := getJsonDataByName(JsonName.Hero)
	if err != nil {
		return nil, err
	}

	var heros []HeroJsonData
	for _, data := range datas {
		if hero, ok := data.(HeroJsonData); ok {
			heros = append(heros, hero)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return heros, nil
}

func GetHeroByID(id string) (HeroJsonData, error) {
	heros, err := GetHeros()
	if err != nil {
		return HeroJsonData{}, err
	}

	for _, hero := range heros {
		if hero.ID == id {
			return hero, nil
		}
	}

	return HeroJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.Hero)
}

// 取得此英雄的3個技能
func (heroJson HeroJsonData) GetSpellJsons() [3]HeroSpellJsonData {
	spellJsons := [3]HeroSpellJsonData{}
	for i := 0; i < 3; i++ {
		spellID := heroJson.GetSpellIDByIdx(i + 1)
		spellJson, err := GetHeroSpellByID(spellID)
		if err != nil {
			fmt.Printf("GetSpellJsons時GetHeroSpellByID(spellID)發生錯誤: %v \n", err)
			continue
		}
		spellJsons[i] = spellJson

	}
	return spellJsons
}

// 傳入X取得英雄第X個技能, EX. 傳入1就是第一個技能
func (heroJson HeroJsonData) GetSpellIDByIdx(idx int) string {
	spellID := heroJson.ID + "_spell" + strconv.Itoa(idx)
	return spellID
}
