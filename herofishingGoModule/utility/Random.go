package utility

import (
	"fmt"
	"math/rand"
	"time"
)

// RandomFloatBetweenInts 從兩個整數之間生成一個隨機float64
func RandomFloatBetweenInts(min, max int) (float64, error) {
	if min > max {
		return 0, fmt.Errorf("RandomFloatBetweenInts傳入值不符合規則 最小值<=最大值")
	}
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	return float64(min) + r.Float64()*(float64(max)-float64(min)), nil
}

// RandomFloatBetweenInts 從兩個整數之間生成一個隨機int
func RandomIntBetweenInts(min, max int) (int, error) {
	if min > max {
		return 0, fmt.Errorf("RandomIntBetweenInts傳入值不符合規則 最小值<=最大值")
	}
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	return r.Intn(max-min+1) + min, nil
}

// GetRandomTFromSlice 傳入泛型切片，返回隨機1個元素。
func GetRandomTFromSlice[T any](slice []T) (T, error) {
	if len(slice) == 0 {
		var value T
		return value, fmt.Errorf("GetRandomTFromSlice傳入參數錯誤")
	}
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	randIndex := r.Intn(len(slice))
	return slice[randIndex], nil
}

// 傳入機率回傳結果 EX. 傳入0.3就是有30%機率返回true
func GetProbResult(prob float64) bool {
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	randomFloat := r.Float64()
	return randomFloat <= prob
}
