package utility

import (
	"math"
	"sync"
)

type accumulator struct {
	keyValueMap map[string]int
	mutex       sync.Mutex
}

// 產生一個新的累加器
func NewAccumulator() *accumulator {
	return &accumulator{
		keyValueMap: make(map[string]int),
	}
}

// 傳入key取得下一個索引編號
func (a *accumulator) GetNextIdx(key string) int {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.keyValueMap[key]; !exists {
		a.keyValueMap[key] = 0
	} else {
		if (a.keyValueMap[key] + 1) <= math.MaxInt {
			a.keyValueMap[key] += 1
		} else {
			a.keyValueMap[key] = 0
		}
	}

	return a.keyValueMap[key]
}
