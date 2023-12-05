package main

import (
	"fmt"
	"herofishingGoModule/gameJson"
)

func main() {
	err := gameJson.Init("Dev")
	if err != nil {
		fmt.Printf("初始化失敗: %v\n", err)
		return
	}
	// hero1, err := gameJson.GetHeroByID("1")
	// if err != nil {
	// 	fmt.Printf("取資料錯誤: %v", err)
	// }

}
