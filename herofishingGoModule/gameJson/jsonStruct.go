package gameJson

import (
	"encoding/json"
	"errors"
	"fmt"
	"herofishingGoModule/logger"

	"github.com/google/martian/v3/log"
)

// jsonDic的結構為jsonDic[jsonName][ID]
var jsonDic = make(map[string]map[string]interface{})

type JsonNameStruct struct {
	GameSetting string
	Hero        string
}

// Json名稱列表
var JsonName = JsonNameStruct{
	GameSetting: "GameSetting",
	Hero:        "Hero",
}

// 傳入Json名稱取得對應JsonMap資料
func getJsonDataByName(name string) (map[string]interface{}, error) {
	data, exists := jsonDic[name]
	if !exists {
		return nil, fmt.Errorf("jsonDic中未找到 %s 的資料", name)
	}
	return data, nil
}

func GetHeros() ([]HeroStruct, error) {
	datas, err := getJsonDataByName(JsonName.Hero)
	if err != nil {
		return nil, err
	}

	var heros []HeroStruct
	for _, data := range datas {
		if hero, ok := data.(HeroStruct); ok {
			heros = append(heros, hero)
		} else {
			return nil, fmt.Errorf("資料類型不匹配，期望 HeroStruct 但得到 %T", data)
		}
	}
	return heros, nil
}

func GetHeroByID(id string) (HeroStruct, error) {
	heros, err := GetHeros()
	if err != nil {
		return HeroStruct{}, err
	}

	for _, hero := range heros {
		if hero.ID == id {
			return hero, nil
		}
	}

	return HeroStruct{}, fmt.Errorf("未找到ID為 %s 的英雄", id)
}

// 傳入Json將並轉為對應struct資料並存入jsonDic中, jsonDic的結構為jsonDic[jsonName][ID]
func SetJsonDic(jsonName string, jsonData []byte) error {

	var unmarshaler JsonUnmarshaler
	switch jsonName {
	case JsonName.GameSetting:
		unmarshaler = GameSettingStruct{}
	case JsonName.Hero:
		unmarshaler = HeroStruct{}
	default:
		return errors.New("未定義的jsonName")
	}
	items, err := unmarshaler.UnmarshalJSONData(jsonData)
	if err != nil {
		log.Errorf("%s Unmarshal失敗: %v", logger.LOG_GameJson, err)
		return err
	}
	jsonDic[jsonName] = items
	return nil
}

type JsonUnmarshaler interface {
	UnmarshalJSONData(jsonData []byte) (map[string]interface{}, error)
}

func (g GameSettingStruct) UnmarshalJSONData(jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]GameSettingStruct
	if err := json.Unmarshal(jsonBytes, &wrapper); err != nil {
		return nil, err
	}

	datas, ok := wrapper["GameSetting"]
	if !ok {
		return nil, fmt.Errorf("未找到 'GameSetting' 鍵")
	}

	items := make(map[string]interface{})
	for _, item := range datas {
		items[item.ID] = item
	}
	return items, nil
}
func (g HeroStruct) UnmarshalJSONData(jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]HeroStruct
	if err := json.Unmarshal(jsonBytes, &wrapper); err != nil {
		return nil, err
	}

	datas, ok := wrapper["Hero"]
	if !ok {
		return nil, fmt.Errorf("未找到 'Hero' 鍵")
	}

	items := make(map[string]interface{})
	for _, item := range datas {
		items[item.ID] = item
	}
	return items, nil
}

// GameSetting JSON
type GameSettingStruct struct {
	ID    string `json:"ID"`
	Value string `json:"Value"`
}

// Hero JSON
type HeroStruct struct {
	ID           string `json:"ID"`
	Ref          string `json:"Ref"`
	RoleCategory string `json:"RoleCategory"`
	IdleMotions  string `json:"IdleMotions"`
}

// HeroEXP JSON
type HeroEXPStruct struct {
	ID  string `json:"ID"`
	EXP string `json:"EXP"`
}

// Map JSON
type MapStruct struct {
	ID                string `json:"ID"`
	Ref               string `json:"Ref"`
	Multiplier        string `json:"Multiplier"`
	MonsterSpawnerIDs string `json:"MonsterSpawnerIDs"`
}

// Monster JSON
type MonsterStruct struct {
	ID           string `json:"ID"`
	Ref          string `json:"Ref"`
	Multiplier   string `json:"Multiplier"`
	EXP          string `json:"EXP"`
	Radius       string `json:"Radius"`
	Speed        string `json:"Speed"`
	MonsterType  string `json:"MonsterType"`
	HitEffectPos string `json:"HitEffectPos"`
}

// MonsterSpawner JSON
type MonsterSpawnerStruct struct {
	ID                      string `json:"ID"`
	SpawnType               string `json:"SpawnType"`
	TypeValue               string `json:"TypeValue"`
	MonsterIDs              string `json:"MonsterIDs"`
	MonsterSpawnIntervalSec string `json:"MonsterSpawnIntervalSec"`
	Routes                  string `json:"Routes"`
}

// Route JSON
type RouteStruct struct {
	ID        string `json:"ID"`
	SpawnPos  string `json:"SpawnPos"`
	TargetPos string `json:"TargetPos"`
}
