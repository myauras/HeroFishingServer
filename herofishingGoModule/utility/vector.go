package utility

import (
	"fmt"
	"math"
)

type Vector2 struct {
	X float64
	Y float64
}

// 取得兩點間的距離
func GetDistance(toPos Vector2, fromPos Vector2) float64 {
	return math.Sqrt(math.Pow(toPos.X-fromPos.X, 2) + math.Pow(toPos.Y-fromPos.Y, 2))
}

// 傳入字串取得Vector2, EX. 傳入"3,2"會回傳(3,2)
func NewVector2(splitedStr string) (Vector2, error) {
	vSlice, err := Split_FLOAT(splitedStr, ",")
	if err != nil {
		return Vector2{}, fmt.Errorf("在NewVector2時Split_FLOAT錯誤: %v", err)
	}
	if len(vSlice) != 2 {
		return Vector2{}, fmt.Errorf("在NewVector2時Split_FLOAT, 結果長度不為2")
	}
	return Vector2{X: vSlice[0], Y: vSlice[1]}, nil
}

// 傳入字串取得Vector2, EX. 傳入"3,1,3"會取X跟Z並回傳(3,3)
func NewVector2XZ(splitedStr string) (Vector2, error) {
	vSlice, err := Split_FLOAT(splitedStr, ",")
	if err != nil {
		return Vector2{}, fmt.Errorf("在NewVector2XZ時Split_FLOAT錯誤: %v", err)
	}
	if len(vSlice) != 3 {
		return Vector2{}, fmt.Errorf("在NewVector2XZ時Split_FLOAT, 結果長度不為3")
	}
	return Vector2{X: vSlice[0], Y: vSlice[2]}, nil
}
