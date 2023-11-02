package utility

import (
	"fmt"
	"math/rand"
	"time"
)

// RandomFloatBetweenInts 從兩個整數之間生成一個隨機浮點數
func RandomFloatBetweenInts(min, max int) (float64, error) {
	if min > max {
		return 0, fmt.Errorf("RandomFloatBetweenInts傳入值不符合規則 最小值<=最大值")
	}

	// 建立一個局部隨機數生成器
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	// 生成一個範圍在 min 到 max 之間的隨機浮點數
	return float64(min) + r.Float64()*(float64(max)-float64(min)), nil
}
