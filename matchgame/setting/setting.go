package setting

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"net"
)

// 玩家
type Player struct {
	ID       string
	Index    int           // 玩家在房間的索引(座位)
	MyHero   *Hero         // 使用中的英雄
	LeftSecs float64       // 玩家已離開遊戲房X秒
	ConnTCP  ConnectionTCP // TCP連線
	ConnUDP  net.Conn      // UDP連線
}

// 英雄
type Hero struct {
	HeroID     int           // 英雄ID
	HeroSkinID string        // SkinID
	HeroEXP    int           // 英雄經驗
	Spells     [3]*HeroSpell // 英雄技能
}

// 英雄技能
type HeroSpell struct {
	SpellJson gameJson.HeroSpellJsonData
	Charge    int // 充能
}

// 將玩家連線斷掉
func (player *Player) CloseConnection() {
	if player == nil {
		return
	}
	if player.ConnTCP.Conn != nil {
		player.ConnTCP.Conn.Close()
		player.ConnTCP.Conn = nil
	}
	if player.ConnUDP != nil {
		player.ConnUDP.Close()
		player.ConnUDP = nil
	}
}

// 取得此英雄隨機尚未充滿能的技能
func (hero *Hero) GetRandomUnchargedSpell() *HeroSpell {
	spells := hero.GetUnchargedSpells()
	if len(spells) == 0 {
		return nil
	}
	spell, err := utility.GetRandomTFromSlice(spells)
	if err != nil {
		log.Errorf("%s utility.GetRandomTFromSlice(spells)錯誤: %v", logger.LOG_Setting, err)
	}
	return spell
}

// 取得此英雄尚未充滿能的技能
func (hero *Hero) GetUnchargedSpells() []*HeroSpell {
	spells := make([]*HeroSpell, 0)
	for _, spell := range hero.Spells {
		if !spell.IsCharged() {
			spells = append(spells, spell)
		}
	}
	return spells
}

// 取得此技能充能比例
func (spell *HeroSpell) GetChargeRatio() float64 {
	return float64(spell.Charge) / float64(spell.SpellJson.Cost)
}

// 此技能是否充滿能
func (spell *HeroSpell) IsCharged() bool {
	return spell.Charge >= spell.SpellJson.Cost
}

type ConnectionTCP struct {
	Conn    net.Conn      // TCP連線
	Encoder *json.Encoder // 連線編碼
	Decoder *json.Decoder // 連線解碼
}

// 伺服器設定
const (
	TIME_UPDATE_INTERVAL_MS        = 100 // 每X毫秒更新Server時間
	AGONES_HEALTH_PIN_INTERVAL_SEC = 2   // 每X秒檢查AgonesServer是否正常運作(官方文件範例是用2秒)
)

// 攻擊事件(包含普攻, 英雄技能, 道具技能, 互動物件等任何攻擊)
// 攻擊事件一段時間清空並存到資料庫中
type AttackEvent struct {
	// 攻擊AttackID格式為 [玩家房間index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	AttackID    string  // 攻擊ID
	ExpiredTime float64 // 過期時間, 房間中的GameTime超過此值就會視為此技能已經結束
	MonsterIdxs [][]int // [波次]-[擊中怪物索引清單]
}
