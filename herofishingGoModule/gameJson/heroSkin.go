package gameJson

import (
	"encoding/json"
	"fmt"
	"herofishingGoModule/utility"
	"strconv"
	"strings"
)

// / HeroSkin JSON
type HeroSkinJsonData struct {
	ID    string `json:"ID"`
	Point string `json:"Point"`
}

func (jsonData HeroSkinJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]HeroSkinJsonData
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

func GetHeroSkins() ([]HeroSkinJsonData, error) {
	datas, err := getJsonDataByName(JsonName.HeroSkin) // Assuming you have JsonName.HeroSkin defined
	if err != nil {
		return nil, err
	}

	var heroSkins []HeroSkinJsonData
	for _, data := range datas {
		if heroSkin, ok := data.(HeroSkinJsonData); ok {
			heroSkins = append(heroSkins, heroSkin)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return heroSkins, nil
}

func GetHeroSkinByID(id string) (HeroSkinJsonData, error) {
	heroSkins, err := GetHeroSkins()
	if err != nil {
		return HeroSkinJsonData{}, err
	}

	for _, heroSkin := range heroSkins {
		if heroSkin.ID == id {
			return heroSkin, nil
		}
	}

	return HeroSkinJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.HeroSkin)
}

// 傳入英雄ID, 取得該英雄的所有SkinIDs
func GetHeroSkinIDsByHeroID(heroID int) ([]string, error) {
	heroSkins, err := GetHeroSkins()
	if err != nil {
		return nil, err
	}
	prefix := strconv.Itoa(heroID) + "_"
	var matches []string
	for _, v := range heroSkins {
		if strings.HasPrefix(v.ID, prefix) {
			matches = append(matches, v.ID)
		}
	}
	return matches, nil
}

// 傳入英雄ID, 取得該英雄的隨機SkinID
func GetRndHeroSkinByHeroID(heroID int) (string, error) {
	skinIDs, err := GetHeroSkinIDsByHeroID(heroID)
	if err != nil {
		return "", err
	}
	rndSkinID, err := utility.GetRandomTFromSlice(skinIDs)
	return rndSkinID, err
}
