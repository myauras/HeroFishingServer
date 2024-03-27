package game

import (
	// "fmt"
	"fmt"
	"herofishingGoModule/gameJson"

	// mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	gSetting "matchgame/setting"
	"time"

	log "github.com/sirupsen/logrus"
)

type Gamer interface {
	GetID() string
	GetIdx() int
	SetIdx(idx int)
	GetHero() *Hero
	GetBuffers() []packet.PlayerBuff
	SetBuffers(buffers []packet.PlayerBuff)
	GetPoint() int
	AddPoint(value int)
	GetGainPoint() int
	AddHeroExp(value int)
	AddSpellCharge(idx int, value int)
	AddDrop(value int)
	RemoveDrop(value int)
	IsOwnedDrop(value int) bool
	GetRandomChargeableSpell() (gameJson.HeroSpellJsonData, bool)
	GetLearnedAndChargeableSpells() []gameJson.HeroSpellJsonData
	CanSpell(idx int) bool
	GetAttackCDBuff() float64
	CloseConnection()
	InitHero(spellLVs [3]int, spellCharges [3]int)
}

// 玩家
type Player struct {
	ID               string                  // DBPlayer的_id
	Point            int                     // 玩家點數
	Drops            [3]int                  // 獲得掉落, 離開遊戲房一定時間後沒回去就會歸0
	PointBuffer      int                     // 溢位點數, 當玩家損失點數是需要補償時, pointBuffer為負的, 反之為正的
	TotalWin         int                     // 玩家歷史總贏點數
	TotalExpenditure int                     // 玩家歷史總花費點數
	RedisPlayer      *redis.RedisPlayer      // RedisDB玩家實例
	Index            int                     // 玩家在房間的索引(座位)
	MyHero           *Hero                   // 使用中的英雄
	GainPoint        int                     // 此玩家在遊戲房總共贏得點數
	LastUpdateAt     time.Time               // 上次收到玩家更新封包(心跳)
	PlayerBuffs      []packet.PlayerBuff     // 玩家Buffers
	LastAttackTime   float64                 // 上次普攻時間
	LastSpellsTime   [3]float64              // 上次施放英雄技能時間
	ConnTCP          *gSetting.ConnectionTCP // TCP連線
	ConnUDP          *gSetting.ConnectionUDP // UDP連線
}

// 初始化玩家英雄
func (player *Player) InitHero(spellLVs [3]int, spellCharges [3]int) {
	if player == nil {
		return
	}

	player.MyHero = &Hero{
		spellLVs:     spellLVs,
		spellCharges: spellCharges,
	}
}

// 取得ID
func (player *Player) GetID() string {
	return player.ID
}

// 設定座位
func (player *Player) SetIdx(idx int) {
	player.Index = idx
}

// 取得座位
func (player *Player) GetIdx() int {
	return player.Index
}

// 取得Hero
func (player *Player) GetHero() *Hero {
	return player.MyHero
}

// 取得玩家Buffers
func (player *Player) GetBuffers() []packet.PlayerBuff {
	return player.PlayerBuffs
}

// 設定玩家Buffers
func (player *Player) SetBuffers(buffers []packet.PlayerBuff) {
	player.PlayerBuffs = buffers
}

// 取得點數
func (player *Player) GetPoint() int {
	return player.Point
}

// 玩家點數增減
func (player *Player) AddPoint(value int) {
	player.RedisPlayer.AddPoint(value)
	player.Point += int(value)

	// 設定玩家本場贏得點數
	if value > 0 {
		player.GainPoint += value
	}
}

// 取得本場遊戲獲得點數
func (player *Player) GetGainPoint() int {
	return player.GainPoint
}

// 取得點數溢位
func (player *Player) GetPTBuffer() int {
	return player.PointBuffer
}

// 玩家點數溢位增減
func (player *Player) AddPTBuffer(value int) {
	player.RedisPlayer.AddPTBuffer(value)
	player.PointBuffer += value
}

// 取得總贏點數
func (player *Player) GetTotalWin() int {
	return player.TotalWin
}

// 玩家總贏點數增減
func (player *Player) AddTotalWin(value int) {
	player.RedisPlayer.AddTotalWin(value)
	player.TotalWin += value
}

// 取得總花費
func (player *Player) GetTotalExpenditure() int {
	return player.TotalExpenditure
}

// 玩家總花費點數增減
func (player *Player) AddTotalExpenditure(value int) {
	player.RedisPlayer.AddTotalExpenditure(value)
	player.TotalExpenditure += value
}

// 英雄經驗增減
func (player *Player) AddHeroExp(value int) {
	player.RedisPlayer.AddHeroExp(value)
	player.MyHero.heroExp += value
}

// 英雄技能升級, idx傳入0~2
func (player *Player) LvUpSpell(idx int) error {
	heroLV, err := gameJson.GetHeroLVByEXP(player.MyHero.heroExp)
	// log.Errorf("heroExp: %v , heroLV: %v usedSpellPoint: %v", player.MyHero.heroExp, heroLV, player.MyHero.GetUsedSpellPoint())

	if err != nil {
		errStr := fmt.Sprintf("%s gameJson.GetHeroLVByEXP錯誤: %v", logger.LOG_Action, err)
		log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}
	remainSpellPoint := int(heroLV) - player.MyHero.GetUsedSpellPoint()
	if remainSpellPoint <= 0 {
		errStr := fmt.Sprintf("%s 技能點數不足 remainSpellPoint: %v", logger.LOG_Action, remainSpellPoint)
		log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}
	if idx < 0 || idx > 2 { // 英雄技能索引只會是1~3
		errStr := fmt.Sprintf("%s 英雄技能索引只會是0~2 spellIdx: %v", logger.LOG_Action, idx)
		log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}
	curLV := player.MyHero.GetSpellLV(idx)
	if curLV > 3 { // SpellLV是0~4, 0是尚未學習,s 4是等級4
		errStr := fmt.Sprintf("%s 該技能索引%v 等級為%v 無法再升級了", logger.LOG_Action, idx, curLV)
		log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}
	// log.Errorf("LvUpSpell idx: %v curLV: %v", idx, curLV)
	player.RedisPlayer.AddSpellLV(idx)
	player.MyHero.AddSpellLV(idx)
	return nil
}

// 技能充能增減, idx傳入0~2
func (player *Player) AddSpellCharge(idx int, value int) {
	if idx < 0 || idx > 2 {
		log.Errorf("%s AddSpellCharge傳入錯誤索引: %v", logger.LOG_Player, idx)
		return
	}
	if value == 0 {
		log.Errorf("%s AddSpellCharge傳入值為0", logger.LOG_Player)
		return
	}
	// log.Errorf("AddSpellCharge idx: %v value: %v", idx, value)
	player.RedisPlayer.AddSpellCharge(idx, value)
	player.MyHero.AddSpellChage(idx, value)
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
	for i, v := range player.Drops {
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
	player.Drops[dropIdx] = value
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
	for i, v := range player.Drops {
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
	player.Drops[dropIdx] = 0
}

// 是否已經擁有此道具
func (player *Player) IsOwnedDrop(value int) bool {
	for _, v := range player.Drops {
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
	log.Infof("%s 關閉玩家(%s)連線", logger.LOG_Player, player.ID)
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
	for i, v := range player.MyHero.spellCharges {
		if player.MyHero.GetSpellLV(i) <= 0 { // 尚未學習的技能就跳過
			continue
		}
		spell, err := player.MyHero.GetSpellJson(i)
		if err != nil {
			log.Errorf("%s GetUnchargedSpells時GetUnchargedSpells錯誤: %v", logger.LOG_Player, err)
			continue
		}
		if v < spell.Cost {
			spells = append(spells, spell)
		}
	}
	return spells
}

// 檢查是否可以施法
func (player *Player) CanSpell(idx int) bool {

	spell, err := player.MyHero.GetSpellJson(idx)
	if err != nil {
		return false
	}
	cost := spell.Cost

	return player.MyHero.GetSpellCharge(idx) >= cost
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
