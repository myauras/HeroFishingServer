package utility

import (
	"fmt"
	"math"
)

type Vector2 struct {
	X float64
	Y float64
}

func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{
		X: v.X + other.X,
		Y: v.Y + other.Y,
	}
}

func (v Vector2) Sub(other Vector2) Vector2 {
	return Vector2{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}
}

// 向量正規化
func (vec Vector2) Normalize() Vector2 {
	mag := math.Sqrt(vec.X*vec.X + vec.Y*vec.Y)
	if mag == 0 {
		return Vector2{X: 0, Y: 0}
	}
	return Vector2{X: vec.X / mag, Y: vec.Y / mag}
}

type Vector3 struct {
	X float64
	Y float64
	Z float64
}

func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

// 向量正規化
func (vec Vector3) Normalize() Vector3 {
	mag := math.Sqrt(vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z)
	if mag == 0 {
		return Vector3{X: 0, Y: 0, Z: 0}
	}
	return Vector3{X: vec.X / mag, Y: vec.Y / mag, Z: vec.Z / mag}
}

func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

type Rect struct {
	Center        Vector2
	Width, Height float64
}

// 取得兩點間的距離
func GetDistance(toPos Vector2, fromPos Vector2) float64 {
	return math.Sqrt(math.Pow(toPos.X-fromPos.X, 2) + math.Pow(toPos.Y-fromPos.Y, 2))
}

// 求兩點間的向量
func Direction(from, to Vector2) Vector2 {
	return Vector2{X: to.X - from.X, Y: to.Y - from.Y}
}

// Lerp計算向量線性插植
func Lerp(start, end Vector2, t float64) Vector2 {
	return Vector2{
		X: start.X + (end.X-start.X)*t,
		Y: start.Y + (end.Y-start.Y)*t,
	}
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
