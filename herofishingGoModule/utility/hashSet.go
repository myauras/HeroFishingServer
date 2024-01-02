package utility

type HashSet map[string]struct{}

func NewHashSet() HashSet {
	return make(map[string]struct{})
}

// Add 向 HashSet 中加元素
func (set HashSet) Add(element string) {
	set[element] = struct{}{}
}

// Remove 從 HashSet 中移除元素
func (set HashSet) Remove(element string) {
	delete(set, element)
}

// Contains 檢查 HashSet 是否包含某個元素
func (set HashSet) Contains(element string) bool {
	_, exists := set[element]
	return exists
}
