package utility

import (
	"strconv"
	"strings"
)

func StrToIntSlice(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	nums := make([]int, 0, len(parts))

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		nums = append(nums, num)
	}

	return nums, nil
}
