package game

import (
	// "fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"herofishingGoModule/utility"
	"matchgame/logger"
	gSetting "matchgame/setting"
	"time"

	log "github.com/sirupsen/logrus"
)

// 玩家
type Player struct {
	DBPlayer     *mongo.DBPlayer         // 玩家DB資料
	RedisPlayer  *redis.RedisPlayer      // RedisDB玩家實例
	Index        int                     // 玩家在房間的索引(座位)
	MyHero       *Hero                   // 使用中的英雄
	GainPoint    int64                   // 此玩家在遊戲房總共贏得點數
	LastUpdateAt time.Time               // 上次收到玩家更新封包(心跳)
	ConnTCP      *gSetting.ConnectionTCP // TCP連線
	ConnUDP      *gSetting.ConnectionUDP // UDP連線
}

// 玩家點數增減
func (player *Player) AddPoint(value int64) {
	player.RedisPlayer.AddPoint(value)
	player.DBPlayer.Point += int64(value)
	player.GainPoint += value
}

// 英雄經驗增減
func (player *Player) AddHeroExp(value int) {
	player.RedisPlayer.AddHeroExp(value)
	player.DBPlayer.HeroExp += value

}

// 技能充能增減, idx傳入1~3
func (player *Player) AddSpellCharge(idx int, value int) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s AddSpellCharge傳入錯誤索引: %v", logger.LOG_Player, idx)
		return
	}
	if value == 0 {
		log.Errorf("%s AddSpellCharge傳入值為0", logger.LOG_Player)
		return
	}
	player.RedisPlayer.AddSpellCharge(idx, value)
	player.DBPlayer.SpellCharges[(idx - 1)] = value
}

// 新增掉落
func (player *Player) AddDrop(value int) {
	if value == 0 {
		log.Errorf("%s AddDrop傳入值為0", logger.LOG_Player)
		return
	}
	if player.IsOwnedDrop(value) {
		log.Errorf("%s AddDrop時已經有此掉落道具, 無法再新增: %v", logger.LOG_Player, value)
		return
	}
	dropIdx := -1
	for i, v := range player.DBPlayer.Drops {
		if v == 0 && v != value {
			dropIdx = i
			break
		}
	}
	if dropIdx == -1 {
		log.Errorf("%s AddDrop時dropIdx為-1", logger.LOG_Player)
		return
	}
	log.Infof("%s 玩家%s獲得Drop idx:%v  dropID:%v", logger.LOG_Player, player.DBPlayer.ID, dropIdx, player.DBPlayer.Drops[dropIdx])
	player.RedisPlayer.SetDrop(dropIdx, value)
	player.DBPlayer.Drops[dropIdx] = value
}

// 移除掉落
func (player *Player) RemoveDrop(value int) {
	if value == 0 {
		log.Errorf("%s RemoveDrop傳入值為0", logger.LOG_Player)
		return
	}
	if !player.IsOwnedDrop(value) {

		return
	}
	dropIdx := -1
	for i, v := range player.DBPlayer.Drops {
		if v == value {
			dropIdx = i
			break
		}
	}
	if dropIdx == -1 {
		log.Errorf("%s RemoveDrop時無此掉落道具, 無法移除: %v", logger.LOG_Player, value)
		log.Errorf("%s RemoveDrop時dropIdx為-1", logger.LOG_Player)
		return
	}
	log.Infof("%s 玩家%s移除Drop idx:%v  dropID:%v", logger.LOG_Player, player.DBPlayer.ID, dropIdx, player.DBPlayer.Drops[dropIdx])
	player.RedisPlayer.SetDrop(dropIdx, 0)
	player.DBPlayer.Drops[dropIdx] = 0
}

// 是否已經擁有此道具
func (player *Player) IsOwnedDrop(value int) bool {
	for _, v := range player.DBPlayer.Drops {
		if v == value {
			return true
		}
	}
	return false
}

// 將玩家連線斷掉
func (player *Player) CloseConnection() {
	if player == nil {
		return
	}
	if player.ConnTCP.Conn != nil {
		player.ConnTCP.Conn.Close()
		player.ConnTCP.Conn = nil
		player.ConnTCP = nil
	}
	if player.ConnUDP.Conn != nil {
		player.ConnUDP.Conn = nil
		player.ConnUDP = nil
	}
}

// 取得此英雄隨機尚未充滿能的技能, 無尚未充滿能的技能時會返回false
func (player *Player) GetRandomUnchargedSpell() (gameJson.HeroSpellJsonData, bool) {
	spells := player.GetUnchargedSpells()
	if len(spells) == 0 {
		return gameJson.HeroSpellJsonData{}, false
	}
	spell, err := utility.GetRandomTFromSlice(spells)
	if err != nil {
		log.Errorf("%s utility.GetRandomTFromSlice(spells)錯誤: %v", logger.LOG_Player, err)
	}
	return spell, true
}

// 取得此英雄尚未充滿能的技能
func (player *Player) GetUnchargedSpells() []gameJson.HeroSpellJsonData {
	spells := make([]gameJson.HeroSpellJsonData, 0)

	for i, v := range player.DBPlayer.SpellCharges {
		spell, err := player.MyHero.GetSpell((i + 1))
		if err != nil {
			log.Errorf("%s GetUnchargedSpells時GetUnchargedSpells錯誤: %v", logger.LOG_Player, err)
			continue
		}
		if v >= spell.Cost {
			spells = append(spells, spell)
		}
	}
	return spells
}

// 檢查是否可以施法
func (player *Player) CanSpell(idx int) bool {

	spell, err := player.MyHero.GetSpell(idx)
	if err != nil {
		return false
	}
	cost := spell.Cost

	return player.DBPlayer.SpellCharges[(idx-1)] >= cost
}
