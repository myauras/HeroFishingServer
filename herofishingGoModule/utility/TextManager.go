package utility

import (
	"fmt"
	"strconv"
	"strings"
)

// 將傳入的字串以傳入的字元分隔並轉為[]int
func Split_INT(str string, char string) ([]int, error) {
	parts := strings.Split(str, char)
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

// 將傳入的字串以傳入的字元分隔並轉為[]float64
func Split_FLOAT(str string, char string) ([]float64, error) {
	parts := strings.Split(str, char)
	nums := make([]float64, 0, len(parts))

	for _, part := range parts {
		num, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, err
		}
		nums = append(nums, num)
	}

	return nums, nil
}

// 將字串最後一個字轉為數字
func ExtractLastDigit(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("ExtractLastDigit時傳入字串為空")
	}
	lastChar := s[len(s)-1:]
	num, err := strconv.Atoi(lastChar)
	if err != nil {
		return 0, fmt.Errorf("ExtractLastDigit最後一個字串 '%s' 不能轉為數字", lastChar)
	}
	return num, nil
}
