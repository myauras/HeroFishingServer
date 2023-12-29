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

// 從一個 slice 中移除指定索引的元素
func RemoveFromSliceByIdx[T any](slice []T, idx int) []T {
	return append(slice[:idx], slice[idx+1:]...)
}

// 從一個 slice 中移除多個索引, 多個索引是來自另外一個slice的元素
func RemoveFromSliceBySlice[T any](slice []T, idxs []int) []T {
	removeSet := make(map[int]bool)
	for _, idx := range idxs {
		removeSet[idx] = true
	}

	var newSlice []T
	for i, v := range slice {
		if !removeSet[i] {
			newSlice = append(newSlice, v)
		}
	}

	return newSlice
}

// Contains 檢查特定元素是否存在於切片中
func Contains[T comparable](slice []T, element T) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}
