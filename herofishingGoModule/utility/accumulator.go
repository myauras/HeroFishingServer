package utility

import (
	"sync"
	"sync/atomic"
)

type AccumulatorStruct struct {
	keyValueMap map[string]*int64
	mutex       sync.Mutex
}

var Accumulator = &AccumulatorStruct{
	keyValueMap: make(map[string]*int64),
}

// 傳入key 與 要累加的value 取得累加後的value
func (accumulator *AccumulatorStruct) GetNextIndex(key string, addValue int64) int64 {
	accumulator.mutex.Lock()
	defer accumulator.mutex.Unlock()

	if _, exists := accumulator.keyValueMap[key]; !exists {
		var initVal int64
		accumulator.keyValueMap[key] = &initVal
	}
	return atomic.AddInt64(accumulator.keyValueMap[key], addValue)
}
