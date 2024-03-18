package game

import (
	"fmt"
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// 取得ID
func AddBot() *Bot {
	newBotIdx := IDAccumulator.GetNextIdx("BotIdx")
	rndHeroJson, err := gameJson.GetRndHero()
	if err != nil {
		log.Errorf("%s gameJson.GetRndHero()錯誤: %v", logger.LOG_BotBehaviour, err)
		return nil
	}
	heroID, err := strconv.ParseInt(rndHeroJson.ID, 64, 64)
	if err != nil {
		log.Errorf("%s strconv.ParseInt(rndHeroJson.ID,64,64)錯誤: %v", logger.LOG_BotBehaviour, err)
		return nil
	}
	heroSkinID, err := gameJson.GetRndHeroSkinByHeroID(int(heroID))
	spellJsons := rndHeroJson.GetSpellJsons()
	hero := Hero{
		ID:     int(heroID),
		SkinID: heroSkinID,
		Spells: spellJsons,
	}
	bot := Bot{
		Index:  newBotIdx,
		MyHero: &hero,
	}
	joined := MyRoom.JoinPlayer(bot)
	if !joined {
		log.Errorf("%s 玩家加入房間失敗", logger.LOG_Main)
		return nil
	}
	return bot
}
