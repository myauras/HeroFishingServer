package game

import (
	// "fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"
	"time"

	log "github.com/sirupsen/logrus"
)

type Gamer interface {
	AddPoint(value int64)
	AddPTBuffer(value int64)
	AddTotalWin(value int64)
	AddTotalExpenditure(value int64)
	AddHeroExp(value int32)
	AddSpellCharge(idx int32, value int32)
	AddDrop(value int32)
	RemoveDrop(value int32)
	IsOwnedDrop(value int32)
	GetRandomChargeableSpell() (gameJson.HeroSpellJsonData, bool)
	GetLearnedAndChargeableSpells() []gameJson.HeroSpellJsonData
	CanSpell(idx int32) bool
	GetAttackCDBuff() float64
}

// 玩家
type Player struct {
	DBPlayer       *mongo.DBPlayer         // 玩家DB資料
	RedisPlayer    *redis.RedisPlayer      // RedisDB玩家實例
	Index          int                     // 玩家在房間的索引(座位)
	MyHero         *Hero                   // 使用中的英雄
	GainPoint      int64                   // 此玩家在遊戲房總共贏得點數
	LastUpdateAt   time.Time               // 上次收到玩家更新封包(心跳)
	PlayerBuffs    []packet.PlayerBuff     // 玩家Buffers
	LastAttackTime float64                 // 上次普攻時間
	LastSpellsTime [3]float64              // 上次施放英雄技能時間
	ConnTCP        *gSetting.ConnectionTCP // TCP連線
	ConnUDP        *gSetting.ConnectionUDP // UDP連線
}

// 玩家點數增減
func (player *Player) AddPoint(value int64) {
	player.RedisPlayer.AddPoint(value)
	player.DBPlayer.Point += int64(value)

}

// 玩家點數溢位增減
func (player *Player) AddPTBuffer(value int64) {
	player.RedisPlayer.AddPTBuffer(value)
	player.DBPlayer.PointBuffer += value
}

// 玩家總贏點數增減
func (player *Player) AddTotalWin(value int64) {
	player.RedisPlayer.AddTotalWin(value)
	player.DBPlayer.TotalWin += value
	player.GainPoint += value
}

// 玩家總花費點數增減
func (player *Player) AddTotalExpenditure(value int64) {
	player.RedisPlayer.AddTotalExpenditure(value)
	player.DBPlayer.TotalExpenditure += value
}

// 英雄經驗增減
func (player *Player) AddHeroExp(value int32) {
	player.RedisPlayer.AddHeroExp(value)
	player.DBPlayer.HeroExp += value

}

// 技能充能增減, idx傳入1~3
func (player *Player) AddSpellCharge(idx int32, value int32) {
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
func (player *Player) AddDrop(value int32) {
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
		if v == 0 {
			dropIdx = i
			break
		}
	}
	if dropIdx == -1 {
		log.Errorf("%s AddDrop時dropIdx為-1", logger.LOG_Player)
		return
	}
	// log.Infof("%s 玩家%s獲得Drop idx:%v  dropID:%v", logger.LOG_Player, player.DBPlayer.ID, dropIdx, player.DBPlayer.Drops[dropIdx])
	player.RedisPlayer.SetDrop(dropIdx, value)
	player.DBPlayer.Drops[dropIdx] = value
}

// 移除掉落
func (player *Player) RemoveDrop(value int32) {
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
	// log.Infof("%s 玩家%s移除Drop idx:%v  dropID:%v", logger.LOG_Player, player.DBPlayer.ID, dropIdx, player.DBPlayer.Drops[dropIdx])
	player.RedisPlayer.SetDrop(dropIdx, 0)
	player.DBPlayer.Drops[dropIdx] = 0
}

// 是否已經擁有此道具
func (player *Player) IsOwnedDrop(value int32) bool {
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
		log.Errorf("%s 關閉玩家連線時 player 為 nil", logger.LOG_Player)
		return
	}
	if player.ConnTCP.Conn != nil {
		player.ConnTCP.MyLoopChan.ClosePackReadStopChan()
		player.ConnTCP.Conn.Close()
		player.ConnTCP.Conn = nil
		player.ConnTCP = nil
	}
	if player.ConnUDP.Conn != nil {
		player.ConnUDP.MyLoopChan.ClosePackReadStopChan()
		player.ConnUDP.Conn = nil
		player.ConnUDP = nil
	}
	log.Infof("%s 關閉玩家(%s)連線", logger.LOG_Player, player.DBPlayer.ID)
}

// 取得此英雄隨機尚未充滿能且已經學習過的技能, 無適合的技能時會返回false
func (player *Player) GetRandomChargeableSpell() (gameJson.HeroSpellJsonData, bool) {
	spells := player.GetLearnedAndChargeableSpells()

	if len(spells) == 0 {
		return gameJson.HeroSpellJsonData{}, false
	}
	spell, err := utility.GetRandomTFromSlice(spells)
	if err != nil {
		log.Errorf("%s utility.GetRandomTFromSlice(spells)錯誤: %v", logger.LOG_Player, err)
		return gameJson.HeroSpellJsonData{}, false
	}
	return spell, true
}

// 取得此英雄所有尚未充滿能且已經學習過的技能
func (player *Player) GetLearnedAndChargeableSpells() []gameJson.HeroSpellJsonData {
	spells := make([]gameJson.HeroSpellJsonData, 0)
	if player == nil {
		return spells
	}
	for i, v := range player.DBPlayer.SpellCharges {
		if player.MyHero.SpellLVs[i+1] <= 0 { // 尚未學習的技能就跳過
			continue
		}
		spell, err := player.MyHero.GetSpell(int32(i + 1))
		if err != nil {
			log.Errorf("%s GetUnchargedSpells時GetUnchargedSpells錯誤: %v", logger.LOG_Player, err)
			continue
		}
		if v < spell.Cost {
			// log.Errorf("已學習且尚未充滿能的技能: %v", spell.ID)
			spells = append(spells, spell)
		}
	}
	return spells
}

// 檢查是否可以施法
func (player *Player) CanSpell(idx int32) bool {

	spell, err := player.MyHero.GetSpell(idx)
	if err != nil {
		return false
	}
	cost := spell.Cost

	return player.DBPlayer.SpellCharges[(idx-1)] >= cost
}

// 取得普攻CD
func (player *Player) GetAttackCDBuff() float64 {
	cdBuff := 1.0
	for _, buff := range player.PlayerBuffs {
		if buff.Name == "Speedup" {
			cdBuff = cdBuff / buff.Value
			break
		}
	}
	return cdBuff
}
