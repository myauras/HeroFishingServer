package game

import (
	// "herofishingGoModule/gameJson"
	"fmt"
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"math"
	"sort"
	"strconv"

	"matchgame/logger"

	"github.com/google/martian/log"
	// log "github.com/sirupsen/logrus"
)

type Monster struct {
	MonsterIdx        int             // 怪物唯一索引, 在怪物被Spawn後由server產生
	ID                int             // 怪物JsonID
	EXP               int             // 怪物經驗
	Odds              int             // 怪物賠率
	DropID            int             // 怪物掉落ID
	DropRTP           int             // 怪物掉落RTP
	SpawnPos          utility.Vector2 // 出生座標
	TargetPos         utility.Vector2 // 目標座標
	Speed             float64         // 移動速度
	Type              string          // 怪物類型
	SpawnTime         float64         // 在遊戲時間第X秒時被產生的
	OutOfBoundaryTime float64         // 在遊戲時間第X秒時被移出邊界
	LeaveTime         float64         // 在遊戲時間第X秒時要被移除
	Effects           []MonsterEffect // 怪物身上狀態
}
type MonsterEffect struct {
	Name    string  // 效果名稱
	Value   float64 // 效果數值
	AtTime  float64 // 開始時間
	EndTime float64 // 結束時間
}

// 此怪物效果是否過期
func (effect MonsterEffect) IsExpired() bool {
	return MyRoom.GameTime > effect.EndTime
}

func GetNewMonster(idx int, spawnPos utility.Vector2, targetPos utility.Vector2, monsterJson gameJson.MonsterJsonData) (*Monster, error) {

	// 取Json資料
	moveSpeed, err := strconv.ParseFloat(monsterJson.Speed, 64)
	if err != nil {
		log.Errorf("%s strconv.ParseFloat(monsterJson.Speed, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
	}
	id, err := strconv.ParseInt(monsterJson.ID, 10, 64)
	if err != nil {
		errLog := fmt.Sprintf("%s strconv.ParseInt(monsterJson.ID, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
		return nil, fmt.Errorf(errLog)
	}

	exp, err := strconv.ParseInt(monsterJson.EXP, 10, 64)
	if err != nil {
		errLog := fmt.Sprintf("%s strconv.ParseInt(monsterJson.EXP, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
		return nil, fmt.Errorf(errLog)
	}
	odds, err := strconv.ParseInt(monsterJson.Odds, 10, 64)
	if err != nil {
		errLog := fmt.Sprintf("%s strconv.ParseInt(monsterJson.Odds, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
		return nil, fmt.Errorf(errLog)
	}
	speed, err := strconv.ParseFloat(monsterJson.Speed, 64)
	if err != nil {
		errLog := fmt.Sprintf("%s strconv.ParseFloat(monsterJson.Speed, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
		return nil, fmt.Errorf(errLog)
	}
	dropID := int64(0)
	dropRTP := int64(0)
	if monsterJson.DropID != "" {
		dropID, err = strconv.ParseInt(monsterJson.DropID, 10, 64)
		if err != nil {
			errLog := fmt.Sprintf("%s strconv.ParseInt(monsterJson.DropID, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
			return nil, fmt.Errorf(errLog)
		}
		dropJson, err := gameJson.GetDropByID(monsterJson.DropID)
		if err != nil {
			errLog := fmt.Sprintf("%s =gameJson.GetDropByID(monsterJson.DropID)錯誤: %v", logger.LOG_MonsterSpawner, err)
			return nil, fmt.Errorf(errLog)
		}
		dropRTP, err = strconv.ParseInt(dropJson.RTP, 10, 64)
		if err != nil {
			errLog := fmt.Sprintf("%s strconv.ParseInt(dropJson.RTP, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
			return nil, fmt.Errorf(errLog)
		}
	}

	// 設定怪物離開時間
	dist := utility.GetDistance(targetPos, spawnPos)
	toTargetTime := dist / moveSpeed
	leaveTime := MyRoom.GameTime + toTargetTime

	// 設定怪物
	monster := &Monster{
		MonsterIdx: idx,
		ID:         int(id),
		EXP:        int(exp),
		Odds:       int(odds),
		DropID:     int(dropID),
		DropRTP:    int(dropRTP),
		SpawnPos:   spawnPos,
		TargetPos:  targetPos,
		Speed:      speed,
		Type:       monsterJson.MonsterType,
		SpawnTime:  MyRoom.GameTime,
		LeaveTime:  leaveTime,
	}

	// 設定怪物出界時間
	err = monster.SetReachBorderTime(utility.Rect{Center: utility.Vector2{X: 0, Y: 0}, Width: 20, Height: 20})
	if err != nil {
		return nil, err
	}

	// 如果怪物產生時有會賦予怪物狀態的場景效果
	for _, sceneEffect := range MyRoom.SceneEffects {
		switch sceneEffect.Name {
		case "Frozen":
			monsterEffect := MonsterEffect{
				Name:    sceneEffect.Name,
				Value:   sceneEffect.Value,
				AtTime:  MyRoom.GameTime,
				EndTime: sceneEffect.EndTime,
			}
			monster.AddEffect(monsterEffect)
		}
	}

	return monster, nil
}

// 怪物是否離開了
func (monster *Monster) IsLeft() bool {
	return MyRoom.GameTime > monster.LeaveTime
}

// 怪物是否出界了
func (monster *Monster) IsOutOfBoundary() bool {
	return MyRoom.GameTime > monster.OutOfBoundaryTime
}

// ※不移除過期狀態, 因為像是取怪物目前座標時, 需要曾經受到過的Frozen效果來計算
// 移除過期的怪物效果
// func (monster *Monster) RemoveExpiredEffects() {
// 	needRemoveEffectIdxs := make([]int, 0)
// 	for i, effect := range monster.Effects {
// 		if effect.IsExpired() {
// 			needRemoveEffectIdxs = append(needRemoveEffectIdxs, i)
// 		}
// 	}
// 	if len(needRemoveEffectIdxs) != 0 {
// 		monster.Effects = utility.RemoveFromSliceByIdxs(monster.Effects, needRemoveEffectIdxs)
// 	}
// }

// 設定怪物出生點到移動出矩形邊界的抵達時間
// 怪物不會超出邊界(速度為0, 或 方向錯誤)時設定回傳error
func (monster *Monster) SetReachBorderTime(rect utility.Rect) error {
	dir := utility.Direction(monster.SpawnPos, monster.TargetPos)
	normalizedDir := dir.Normalize()
	// 計算各軸時間單位位移量
	speedX := normalizedDir.X * monster.Speed
	speedY := normalizedDir.Y * monster.Speed
	if speedX == 0 && speedY == 0 {
		return fmt.Errorf("怪物單位移動速度為0")
	}

	// 計算邊界
	leftBoundary := rect.Center.X - rect.Width/2
	rightBoundary := rect.Center.X + rect.Width/2
	topBoundary := rect.Center.Y + rect.Height/2
	bottomBoundary := rect.Center.Y - rect.Height/2

	// 定義最短時間
	minTime := math.MaxFloat64

	// 檢查並計算到達每個邊界需求時間
	if speedX != 0 {
		if speedX > 0 { // 往右走就只考慮右邊的邊界
			rightTime := (rightBoundary - monster.SpawnPos.X) / speedX
			if rightTime > 0 && rightTime < minTime {
				minTime = rightTime
			}
		} else { // 往左走只考慮左邊邊界
			leftTime := (leftBoundary - monster.SpawnPos.X) / speedX
			if leftTime > 0 && leftTime < minTime {
				minTime = leftTime
			}
		}
	}
	if speedY != 0 {
		if speedY > 0 { // 往上走就只考慮上方邊界
			topTime := (topBoundary - monster.SpawnPos.Y) / speedY
			if topTime > 0 && topTime < minTime {
				minTime = topTime
			}
		} else { // 往下走就只考慮下方邊界
			bottomTime := (bottomBoundary - monster.SpawnPos.Y) / speedY
			if bottomTime > 0 && bottomTime < minTime {
				minTime = bottomTime
			}
		}
	}

	// 如果沒有在限制時間內超過邊界就回傳-1
	if minTime == math.MaxFloat64 {
		return fmt.Errorf("怪物移動方向錯誤")
	}
	monster.OutOfBoundaryTime = minTime + monster.SpawnTime
	return nil
}

// 取得怪物目前位置
func (monster *Monster) GetVec2Pos() utility.Vector2 {
	moveTime := MyRoom.GameTime - monster.SpawnTime                   // 移動時間
	frozenTime := monster.calculateTotalEffectAvailableTime("Frozen") // 取得Frozen效果的有效影響時間
	moveTime -= frozenTime                                            // 實際移動時間要扣掉被冰住的時間

	distance := utility.GetDistance(monster.TargetPos, monster.SpawnPos) // 總距離
	moveDistance := moveTime * monster.Speed                             // 實際移動距離
	progress := moveDistance / distance                                  // 移動比例

	// Lerp計算向量線性插植
	return utility.Lerp(monster.SpawnPos, monster.TargetPos, progress)
}

// 取得怪物目前位置
func (monster *Monster) GetVec3Pos() utility.Vector3 {
	vec2Pos := monster.GetVec2Pos()
	return utility.Vector3{X: vec2Pos.X, Y: 0, Z: vec2Pos.Y}
}

// 根據AtTime對MonsterEffect進行排序
type SortByAtTime []MonsterEffect

func (a SortByAtTime) Len() int           { return len(a) }
func (a SortByAtTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByAtTime) Less(i, j int) bool { return a[i].AtTime < a[j].AtTime }

// 計算MonsterEffect清單中某Effect, 實際有效影響該怪物的時間
func (monster *Monster) calculateTotalEffectAvailableTime(effectName string) float64 {
	// 取目標效果
	effects := make([]MonsterEffect, 0)
	for _, effect := range monster.Effects {
		if effect.Name == effectName {
			effects = append(effects, effect)
		}
	}

	// 對目標效果做AtTime排序
	sort.Sort(SortByAtTime(effects))

	var totalTime float64
	var curEnd float64

	for i, effect := range effects {
		if i == 0 {
			curEnd = effect.EndTime
			continue
		}

		if effect.AtTime > curEnd {
			totalTime += curEnd - effects[i-1].AtTime
			curEnd = effect.EndTime
		} else if effect.EndTime > curEnd {
			curEnd = effect.EndTime
		}

		if i == len(effects)-1 {
			totalTime += curEnd - effect.AtTime
		}
	}

	return totalTime
}

// 增加效果
func (monster *Monster) AddEffect(effect MonsterEffect) {
	monster.Effects = append(monster.Effects, effect)
	duration := effect.EndTime - MyRoom.GameTime
	// 賦予冰凍效果時, 怪物的離開時間與抵達邊界時間也會延長
	if effect.Name == "Forzen" {
		monster.OutOfBoundaryTime += duration
		monster.LeaveTime += duration
	}
}
