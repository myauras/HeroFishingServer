package utility

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

func RemoveFromMapByKeys[K comparable, V any](myMap map[K]*V, keys []K) {
	for _, key := range keys {
		delete(myMap, key)
	}
}
