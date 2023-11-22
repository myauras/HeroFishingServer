package utility

import (
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

// 傳入key 與 要累加的value 取得累加後的value
func (a *accumulator) GetNextIndex(key string, addValue int) int {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.keyValueMap[key]; !exists {
		a.keyValueMap[key] = 0
	}

	a.keyValueMap[key] += addValue
	return a.keyValueMap[key]
}
