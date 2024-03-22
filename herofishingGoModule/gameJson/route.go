package gameJson

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/logger"
	"herofishingGoModule/utility"
)

// Route JSON
type RouteJsonData struct {
	ID        string `json:"ID"`
	SpawnPos  string `json:"SpawnPos"`
	TargetPos string `json:"TargetPos"`
}

func (jsonData RouteJsonData) UnmarshalJSONData(jsonName string, jsonBytes []byte) (map[string]interface{}, error) {
	var wrapper map[string][]RouteJsonData
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

func GetRoutes() ([]RouteJsonData, error) {
	datas, err := getJsonDataByName(JsonName.Route)
	if err != nil {
		return nil, err
	}

	var routes []RouteJsonData
	for _, data := range datas {
		if route, ok := data.(RouteJsonData); ok {
			routes = append(routes, route)
		} else {
			return nil, fmt.Errorf("資料類型不匹配: %T", data)
		}
	}
	return routes, nil
}

func GetRouteByID(id string) (RouteJsonData, error) {
	routes, err := GetRoutes()
	if err != nil {
		return RouteJsonData{}, err
	}

	for _, route := range routes {
		if route.ID == id {
			return route, nil
		}
	}

	return RouteJsonData{}, fmt.Errorf("未找到ID為 %s 的%s資料", id, JsonName.Route)
}

func (json RouteJsonData) GetSpawnPosAndEndPos() (utility.Vector2, utility.Vector2) {
	spawnPos, err := utility.Split_FLOAT(json.SpawnPos, ",")
	if err != nil {
		log.Errorf("%s utility.Split_FLOAT錯誤 JsonID為: %s", logger.LOG_GameJson, json.ID)
		return utility.Vector2{}, utility.Vector2{}
	}
	targetPos, err := utility.Split_FLOAT(json.TargetPos, ",")
	if err != nil {
		log.Errorf("%s utility.Split_FLOAT錯誤 JsonID為: %s", logger.LOG_GameJson, json.ID)
		return utility.Vector2{}, utility.Vector2{}
	}
	if len(spawnPos) != 3 || len(targetPos) != 3 {
		log.Errorf("%s route表格出錯, spawnPos或targetPos長度不為3 JsonID為: %s", logger.LOG_GameJson, json.ID)
		return utility.Vector2{}, utility.Vector2{}
	}
	return utility.Vector2{X: spawnPos[0], Y: spawnPos[2]}, utility.Vector2{X: targetPos[0], Y: targetPos[2]}
}
