package utility

type Number interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64
}

// 移除重複的元素
func RemoveDuplicatesFromSlice[T comparable](slice []T) []T {
	unique := make(map[T]bool)
	var result []T

	for _, v := range slice {
		if _, ok := unique[v]; !ok {
			unique[v] = true
			result = append(result, v)
		}
	}
	return result
}

// 計算數字切片的總和
func SliceSum[T Number](slice []T) T {
	var sum T
	for _, v := range slice {
		sum += v
	}
	return sum
}

// 從map中移除傳入的key陣列
func RemoveFromMapByKeys[K comparable, V any](myMap map[K]*V, keys []K) {
	for _, key := range keys {
		delete(myMap, key)
	}
}
