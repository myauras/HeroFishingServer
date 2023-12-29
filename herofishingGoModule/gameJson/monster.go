package gameJson

import (
	"encoding/json"
	"fmt"
	// "herofishingGoModule/logger"
	// "herofishingGoModule/utility"
	// "strconv"
	// "github.com/google/martian/log"
)

// Monster JSON
type MonsterJsonData struct {
	ID     string `json:"ID"`
	Ref    string `json:"Ref"`
	Odds   string `json:"Odds"`
	EXP    string `json:"EXP"`
	DropID string `json:"DropID"`
	// Radius       string `json:"Radius"`
	Speed       string `json:"Speed"`
	MonsterType string `json:"MonsterType"`
	// HitEffectPos string `json:"HitEffectPos"`
}

func (jsonData MonsterJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]MonsterJsonData
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

func GetMonsters() ([]MonsterJsonData, error) {
	datas, err := getJsonDataByName(JsonName.Monster)
	if err != nil {
		return nil, err
	}

	var monsters []MonsterJsonData
	for _, data := range datas {
		if monster, ok := data.(MonsterJsonData); ok {
			monsters = append(monsters, monster)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return monsters, nil
}

func GetMonsterByID(id string) (MonsterJsonData, error) {
	monsters, err := GetMonsters()
	if err != nil {
		return MonsterJsonData{}, err
	}

	for _, monster := range monsters {
		if monster.ID == id {
			return monster, nil
		}
	}

	return MonsterJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.Monster)
}

// // 取得掉落IDs(DropIDs改DropID, 不能填多個掉落)
// func (mJson MonsterJsonData) GetDropIds() []int {
// 	if mJson.DropID == "" {
// 		return nil
// 	}
// 	ids, err := utility.StrToIntSlice(mJson.DropID, ",")
// 	if err != nil {
// 		log.Errorf("%s 怪物表(ID: %s)的DropIDs欄位格式錯誤: %v", logger.LOG_GameJson, mJson.ID, err)
// 		return nil
// 	}
// 	return ids
// }

// // 取得DropJsonDatas(DropIDs改DropID, 不能填多個掉落)
// func (mJson MonsterJsonData) GetDropJsonDatas() []DropJsonData {
// 	dropIds := mJson.GetDropIds()
// 	dropJsonDatas := make([]DropJsonData, 0)
// 	for _, v := range dropIds {
// 		dropJson, err := GetDropByID(strconv.Itoa(v))
// 		if err != nil {
// 			log.Errorf("%s GetDropByID(strconv.Itoa(v)) v: %s 發生錯誤: %v", logger.LOG_GameJson, v, err)
// 			continue
// 		}
// 		dropJsonDatas = append(dropJsonDatas, dropJson)
// 	}
// 	return dropJsonDatas
// }
