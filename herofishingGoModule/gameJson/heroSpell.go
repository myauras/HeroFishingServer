package gameJson

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/logger"
	"herofishingGoModule/utility"
	"strconv"
)

// HeroSpell JSON
type HeroSpellJsonData struct {
	ID              string    `json:"ID"`
	RTP             []float64 `json:"RTP"`
	CD              float64   `json:"CD"`
	Cost            int       `json:"Cost"`
	MaxHits         int       `json:"MaxHits"`
	SpellType       string    `json:"SpellType"`
	SpellTypeValues string    `json:"SpellTypeValues"`
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

		rtps, err := utility.Split_FLOAT(aux.RTP, ",")
		if err != nil {
			return err
		}
		spellJson.RTP = rtps
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
		spellJson.Cost = int(intVal)
	}
	if aux.MaxHits != "" {
		var intVal int64
		if intVal, err = strconv.ParseInt(aux.MaxHits, 10, 32); err != nil {
			return err
		}
		spellJson.MaxHits = int(intVal)
	}

	return nil
}

// 此HeroSpellJsonData的技能類型, "Attack":普攻 "HeroSpell":英雄技能 "DropSpell":掉落技能
func (heroSpellJsonData HeroSpellJsonData) GetSpellType() string {
	if len(heroSpellJsonData.RTP) == 0 {
		return "Attack"
	}
	lastChar := heroSpellJsonData.ID[len(heroSpellJsonData.ID)-1:]
	_, err := strconv.Atoi(lastChar)
	if err == nil {
		return "HeroSpell"
	}

	return "DropSpell"
}

// 取得該技能等級對應的RTP, 等級傳入1就是等級1的技能, 如果該技能沒有該等級的RTP就會回傳0
func (heroSpellJsonData HeroSpellJsonData) GetRTP(lv int) float64 {
	lv -= 1
	if lv < 0 || lv >= len(heroSpellJsonData.RTP) {
		log.Error("玩家尚未學習技能卻施放該技能")
		return heroSpellJsonData.RTP[0]
		// return 0
	}
	return heroSpellJsonData.RTP[lv]
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

// 返回0代表立即
func (heroSpellJsonData HeroSpellJsonData) GetSpellSpeed() float64 {
	// heroSpellJsonData.SpellTypeValues第X個數值代表什麼意思可以參考HeroSpell表的註解
	values, err := utility.Split_FLOAT(heroSpellJsonData.SpellTypeValues, ",")
	if err != nil {
		log.Errorf("%s GetSpellSpeed時,  utility.Split_FLOAT錯誤: %s", logger.LOG_GameJson, err)
		return 0
	}
	switch heroSpellJsonData.SpellType {
	case "LineShot":
		if len(values) < 2 {
			log.Errorf("%s GetSpellSpeed時,  SpellTypeValues錯誤 SpellType: %s", logger.LOG_GameJson, heroSpellJsonData.SpellType)
			return 0
		}
		return values[2]
	case "SpreadLineShot":
		if len(values) < 2 {
			log.Errorf("%s GetSpellSpeed時,  SpellTypeValues錯誤 SpellType: %s", logger.LOG_GameJson, heroSpellJsonData.SpellType)
			return values[2]
		}
		return 0
	case "CircleArea":
		return 0
	case "SectorArea":
		return 0
	default:
		log.Errorf("%s GetSpellSpeed時, 尚未定義的heroSpellJsonData.SpellTyp: %s", logger.LOG_GameJson, heroSpellJsonData.SpellType)
		return 0
	}
}
