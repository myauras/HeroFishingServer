package game

import (
	// "herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"math"

	"github.com/google/martian/log"
	"matchgame/logger"
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
	SpawnTime         float64         // 在遊戲時間第X秒時被產生的
	OutOfBoundaryTime float64         // 在遊戲時間第X秒時被移出邊界
	LeaveTime         float64         // 在遊戲時間第X秒時要被移除
}

// 怪物是否離開了
func (monster *Monster) IsLeft() bool {
	return MyRoom.GameTime > monster.LeaveTime
}

// 怪物是否出界了
func (monster *Monster) IsOutOfBoundary() bool {
	return MyRoom.GameTime > monster.OutOfBoundaryTime
}

// 取得怪物目前位置
func (monster *Monster) GetCurPos() utility.Vector2 {
	moveTime := MyRoom.GameTime - monster.SpawnTime                      // 移動時間
	distance := utility.GetDistance(monster.TargetPos, monster.SpawnPos) // 總距離
	moveDistance := moveTime * monster.Speed                             // 實際移動距離
	progress := moveDistance / distance                                  // 移動比例

	// Lerp計算向量線性插植
	return utility.Lerp(monster.SpawnPos, monster.TargetPos, progress)
}

// 計算矩形座標內的怪物是否能在時間內移動出矩形邊界
// 可以的話回傳遊戲時間第幾秒時移出邊界, 否則回傳-1代表怪物永遠不會超出邊界(速度為0 或 時間內無法達到)
// limitTime傳入-1代表不限制時間
func (monster *Monster) GetReachBorderTime(rect utility.Rect, limitTime float64) float64 {
	dir := utility.Direction(monster.SpawnPos, monster.TargetPos)
	normalizedDir := utility.Normalize(dir)
	// 計算各軸時間單位位移量
	speedX := normalizedDir.X * monster.Speed
	speedY := normalizedDir.Y * monster.Speed
	if speedX == 0 && speedY == 0 {
		log.Errorf("%v 怪物單位移動速度為0", logger.LOG_Monster)
		return -1
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
	if limitTime > 0 {
		if minTime == math.MaxFloat64 || minTime+monster.SpawnTime > limitTime {
			return -1
		}
	} else {
		if minTime == math.MaxFloat64 {
			return -1
		}
	}
	return minTime + monster.SpawnTime
}
