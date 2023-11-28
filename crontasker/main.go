package main

import (
	logger "crontasker/logger"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

const (
	PLAYER_OFFLINE_CRON = "*/3 * * * *"
)

func main() {
	myCron := cron.New()
	_, err := myCron.AddFunc(PLAYER_OFFLINE_CRON, playerOfflineHandler)
	if err != nil {
		log.Infof("%s 安排playerOfflineHandler錯誤: %v \n", logger.LOG_Main, err)
		return
	}
	myCron.Start()

	select {}
}

func playerOfflineHandler() {
	log.Infof("%s 處理玩家離線 \n", logger.LOG_Main)
}
