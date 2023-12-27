package redis

import (
	"fmt"
	"sync"
	"time"

	logger "herofishingGoModule/logger"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

var dbWriteMinMiliSecs = 1000

var players map[string]*RedisPlayer

type RedisPlayer struct {
	id                 string // Redis的PlayerID是"player-"+mongodb player id, 例如player-6538c6f219a12eb9e4ded943
	pointBuffer        int64  // 暫存點數修改
	heroExpBuffer      int    // 暫存經驗修改
	spellChargesBuffer [3]int // 暫存技能充能
	dropsBuffer        [3]int // 暫存掉落道具
	updateOn           bool   // 資料定時更新上RedisDB程序開關
	MutexLock          sync.Mutex
}
type DBPlayer struct {
	ID           string `redis:"id"`
	Point        int64  `redis:"point"`        // 點數
	HeroExp      int    `redis:"heroExp"`      // 英雄經驗
	SpellCharge1 int    `redis:"spellCharge1"` // 技能充能1
	SpellCharge2 int    `redis:"spellCharge2"` // 技能充能2
	SpellCharge3 int    `redis:"spellCharge3"` // 技能充能3
	DropCharge1  int    `redis:"dropCharge1"`  // 掉落道具1
	DropCharge2  int    `redis:"dropCharge2"`  // 掉落道具2
	DropCharge3  int    `redis:"dropCharge3"`  // 掉落道具3
}

// 將暫存的數據寫入RedisDB
func (rPlayer *RedisPlayer) WritePlayerUpdateToRedis() {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	if rPlayer.pointBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "point", int64(rPlayer.pointBuffer)).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis point錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.pointBuffer = 0
	}
	if rPlayer.heroExpBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "heroExp", int64(rPlayer.heroExpBuffer)).Result() // 轉換為 int64
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis heroExp錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.heroExpBuffer = 0
	}
	for i, charge := range rPlayer.spellChargesBuffer {
		if charge != 0 {
			_, err := rdb.HSet(ctx, rPlayer.id, fmt.Sprintf("spellCharge%d", (i+1)), int64(charge)).Result()
			if err != nil {
				log.Errorf("%s writePlayerUpdateToRedis spellCharge錯誤: %v", logger.LOG_Redis, err)
			}
			rPlayer.spellChargesBuffer[i] = 0
		}
	}
	for i, drop := range rPlayer.dropsBuffer {
		if drop != 0 {
			_, err := rdb.HSet(ctx, rPlayer.id, fmt.Sprintf("drop%d", (i+1)), int64(drop)).Result()
			if err != nil {
				log.Errorf("%s writePlayerUpdateToRedis drop錯誤: %v", logger.LOG_Redis, err)
			}
			rPlayer.dropsBuffer[i] = 0
		}
	}

}

// 關閉玩家
func ClosePlayer(playerID string) {
	if _, ok := players[playerID]; ok {
		players[playerID].StopInGameUpdatePlayer()
		players[playerID].WritePlayerUpdateToRedis()
		delete(players, playerID) // 從 map 中移除
	} else {
		log.Errorf("%s ClosePlayer錯誤 玩家 %s 不存在map中", logger.LOG_Redis, playerID)
		return
	}
}

// 關閉玩家
func (player *RedisPlayer) ClosePlayer() {
	ClosePlayer(player.id)
}

// 建立玩家資料
func CreatePlayerData(playerID string, point int, heroExp int) (*RedisPlayer, error) {
	playerID = "player-" + playerID

	dbPlayer, err := GetPlayerDBData(playerID)
	if err != nil || dbPlayer.ID == "" {
		// 建立玩家RedisDB資料
		_, err := rdb.HMSet(ctx, playerID, map[string]interface{}{
			"id":           playerID,
			"point":        point,
			"heroExp":      heroExp,
			"spellCharge1": 0,
			"spellCharge2": 0,
			"spellCharge3": 0,
			"drop1":        0,
			"drop2":        0,
			"drop3":        0,
		}).Result()
		if err != nil {
			return nil, fmt.Errorf("%s createPlayerData錯誤: %v", logger.LOG_Redis, err)
		}

	}

	player := RedisPlayer{
		id:                 playerID,
		spellChargesBuffer: [3]int{0, 0, 0},
		dropsBuffer:        [3]int{0, 0, 0},
		updateOn:           true,
	}
	go player.updatePlayer()

	if _, ok := players[playerID]; !ok {
		players[playerID] = &player
	} else {
		return nil, fmt.Errorf("%s createPlayerData錯誤 玩家 %s 已存在map中", logger.LOG_Redis, playerID)
	}
	return &player, nil
}

// 開始跑玩家資料定時更新上RedisDB程序
func (rPlayer *RedisPlayer) StartInGameUpdatePlayer() {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.updateOn = true
}

// 停止跑玩家資料定時更新上RedisDB程序
func (rPlayer *RedisPlayer) StopInGameUpdatePlayer() {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.updateOn = false
}

// 增加點數
func (rPlayer *RedisPlayer) AddPoint(value int64) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.pointBuffer += value
}

// 增加英雄經驗
func (rPlayer *RedisPlayer) AddHeroExp(value int) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.heroExpBuffer += value
}


// 設定技能充能, idx傳入1~3
func (rPlayer *RedisPlayer) AddSpellCharge(idx int, value int) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s AddSpellCharge傳入錯誤索引: %v", logger.LOG_Redis, idx)
		return
	}
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.spellChargesBuffer[(idx - 1)] += value
}

// 設定掉落道具
func (rPlayer *RedisPlayer) AddDrop(value int) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	for i, v := range rPlayer.dropsBuffer {
		if v == 0 {
			rPlayer.dropsBuffer[i] = value
			break
		}
	}
}

// 暫存資料寫入並每X毫秒更新上RedisDB
func (player *RedisPlayer) updatePlayer() {
	ticker := time.NewTicker(time.Duration(dbWriteMinMiliSecs) * time.Millisecond)
	defer ticker.Stop()
	for {
		if !player.updateOn {
			continue
		}
		select {
		case <-ticker.C:
			player.WritePlayerUpdateToRedis()
		case <-ctx.Done():
			return
		}
	}
}

// 取得RedisDB中Player資料
func (player *RedisPlayer) GetPlayerDBData() {
	GetPlayerDBData(player.id)
}

// 取得RedisDB中Player資料, 找不到玩家資料時DBPlayer會返回0值
func GetPlayerDBData(playerID string) (DBPlayer, error) {
	var player DBPlayer
	val, err := rdb.HGetAll(ctx, playerID).Result()
	if err != nil {
		return player, fmt.Errorf("ShowPlayer錯誤: %v", err)
	}
	if len(val) == 0 { // 找不到資料回傳0值
		return player, nil
	}
	err = mapstructure.Decode(val, &player)
	if err != nil {
		return player, fmt.Errorf("RedisDB Plaeyr 反序列化錯誤: %v", err)
	}
	// log.Infof("%s playerID: %s point: %d heroExp: %d\n", logger.LOG_Redis, player.ID, player.Point, player.HeroExp)
	return player, nil

}
