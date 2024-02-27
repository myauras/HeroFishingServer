package redis

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	logger "herofishingGoModule/logger"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

var dbWriteMinMiliSecs = 1000

var players map[string]*RedisPlayer

type RedisPlayer struct {
	id                     string    // Redis的PlayerID是"player-"+mongodb player id, 例如player-6538c6f219a12eb9e4ded943
	pointBuffer            int64     // 暫存點數修改
	ptBufferBuffer         int64     // 暫存點數溢位修改
	totalWinBuffer         int64     // 暫存總贏點數修改
	totalExpenditureBuffer int64     // 暫存總花費點數修改
	heroExpBuffer          int32     // 暫存經驗修改
	spellChargesBuffer     [3]int32  // 暫存技能充能
	dropsBuffer            [3]int32  // 暫存掉落道具
	MyLoopChan             *LoopChan // 資料更新關閉通道
	MutexLock              sync.Mutex
}

// 關閉PackReadStopChan通道
func (loopChan *LoopChan) closePackReadStopChan() {
	loopChan.ChanCloseOnce.Do(func() {
		close(loopChan.StopChan)
	})
}

type LoopChan struct {
	StopChan      chan struct{}
	ChanCloseOnce sync.Once
}

// ※注意: 因為RedisDB都是存成字串, 所以有新增非字串的類型要定義DecodeHook
type RedisDBPlayer struct {
	ID               string `redis:"id"`
	Point            int64  `redis:"point"`             // 點數
	PointBuffer      int64  `redis:"pointBuffer"`       // 點數
	TotalWin         int64  `redis:"totalWin"`          // 總贏點數
	TotalExpenditure int64  `redis:"totalExpenditure "` // 總花費點數
	HeroExp          int32  `redis:"heroExp"`           // 英雄經驗
	SpellCharge1     int32  `redis:"spellCharge1"`      // 技能充能1
	SpellCharge2     int32  `redis:"spellCharge2"`      // 技能充能2
	SpellCharge3     int32  `redis:"spellCharge3"`      // 技能充能3
	Drop1            int32  `redis:"drop1"`             // 掉落道具1
	Drop2            int32  `redis:"drop2"`             // 掉落道具2
	Drop3            int32  `redis:"drop3"`             // 掉落道具3
}

// 定義DecodeHook，將特定字串轉換為指定類型
var decodeHook = mapstructure.ComposeDecodeHookFunc(
	mapstructure.StringToSliceHookFunc(","),
	mapstructure.StringToTimeHookFunc(time.RFC3339),
	func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() == reflect.String {
			strData := data.(string)
			switch t {
			case reflect.TypeOf(int(0)):
				parsed, err := strconv.ParseInt(strData, 10, 64)
				return int(parsed), err
			case reflect.TypeOf(int32(0)):
				parsed, err := strconv.ParseInt(strData, 10, 32)
				return int32(parsed), err
			case reflect.TypeOf(int64(0)):
				return strconv.ParseInt(strData, 10, 64)
			}
		}
		return data, nil
	},
)

// 將暫存的數據寫入RedisDB
func (rPlayer *RedisPlayer) WritePlayerUpdateToRedis() {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	if rPlayer.pointBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "point", rPlayer.pointBuffer).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis point錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.pointBuffer = 0
	}
	if rPlayer.ptBufferBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "pointBuffer", rPlayer.ptBufferBuffer).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis pointBuffer錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.ptBufferBuffer = 0
	}
	if rPlayer.totalWinBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "totalWinB", rPlayer.totalWinBuffer).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis totalWinB錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.totalWinBuffer = 0
	}
	if rPlayer.totalExpenditureBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "totalExpenditure", rPlayer.totalExpenditureBuffer).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis totalExpenditure錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.totalExpenditureBuffer = 0
	}
	if rPlayer.heroExpBuffer != 0 {
		_, err := rdb.HIncrBy(ctx, rPlayer.id, "heroExp", int64(rPlayer.heroExpBuffer)).Result()
		if err != nil {
			log.Errorf("%s writePlayerUpdateToRedis heroExp錯誤: %v", logger.LOG_Redis, err)
		}
		rPlayer.heroExpBuffer = 0
	}
	for i, charge := range rPlayer.spellChargesBuffer {
		if charge != 0 {
			_, err := rdb.HSet(ctx, rPlayer.id, fmt.Sprintf("spellCharge%d", (i+1)), charge).Result()
			if err != nil {
				log.Errorf("%s writePlayerUpdateToRedis spellCharge錯誤: %v", logger.LOG_Redis, err)
			}
			rPlayer.spellChargesBuffer[i] = 0
		}
	}
	for i, drop := range rPlayer.dropsBuffer {
		if drop != 0 {
			_, err := rdb.HSet(ctx, rPlayer.id, fmt.Sprintf("drop%d", (i+1)), drop).Result()
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
		players[playerID].MyLoopChan.closePackReadStopChan()
		players[playerID].WritePlayerUpdateToRedis()
		delete(players, playerID)
		log.Infof("%s 移除Redis Player(%s)", logger.LOG_Redis, playerID)
	} else {
		log.Warnf("%s ClosePlayer錯誤 玩家 %s 不存在map中", logger.LOG_Redis, playerID)
		return
	}
}

// 關閉玩家
func (player *RedisPlayer) ClosePlayer() {
	ClosePlayer(player.id)
}

// 建立玩家資料
func CreatePlayerData(playerID string, point int64, ptBuffer int64, totalWin int64, totalExpenditure int64, heroExp int32, spellCharges [3]int32, drops [3]int32) (*RedisPlayer, error) {
	playerID = "player-" + playerID

	dbPlayer, err := GetPlayerDBData(playerID)
	if err != nil || dbPlayer.ID == "" {
		// 建立玩家RedisDB資料
		_, err := rdb.HMSet(ctx, playerID, map[string]interface{}{
			"id":               playerID,
			"point":            point,
			"pointBuffer":      ptBuffer,
			"totalWin":         totalWin,
			"totalExpenditure": totalExpenditure,
			"heroExp":          heroExp,
			"spellCharge1":     spellCharges[0],
			"spellCharge2":     spellCharges[1],
			"spellCharge3":     spellCharges[2],
			"drop1":            drops[0],
			"drop2":            drops[1],
			"drop3":            drops[2],
		}).Result()
		if err != nil {
			return nil, fmt.Errorf("%s createPlayerData錯誤: %v", logger.LOG_Redis, err)
		}

	}

	player := RedisPlayer{
		id:                 playerID,
		spellChargesBuffer: [3]int32{0, 0, 0},
		dropsBuffer:        [3]int32{0, 0, 0},
		MyLoopChan: &LoopChan{
			StopChan: make(chan struct{}, 1),
		},
	}
	go player.updatePlayer()

	if _, ok := players[playerID]; !ok {
		players[playerID] = &player
	} else {
		return nil, fmt.Errorf("%s createPlayerData錯誤 玩家 %s 已存在map中", logger.LOG_Redis, playerID)
	}
	log.Infof("%s 建立Redis Player(%s)", logger.LOG_Redis, playerID)
	return &player, nil
}

// 增加點數
func (rPlayer *RedisPlayer) AddPoint(value int64) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.pointBuffer += value
}

// 增加點數溢位
func (rPlayer *RedisPlayer) AddPTBuffer(value int64) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.ptBufferBuffer += value
}

// 增加總贏點數
func (rPlayer *RedisPlayer) AddTotalWin(value int64) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.totalWinBuffer += value
}

// 增加總花費點數
func (rPlayer *RedisPlayer) AddTotalExpenditure(value int64) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.totalExpenditureBuffer += value
}

// 增加英雄經驗
func (rPlayer *RedisPlayer) AddHeroExp(value int32) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.heroExpBuffer += value
}

// 設定技能充能, idx傳入1~3
func (rPlayer *RedisPlayer) AddSpellCharge(idx int32, value int32) {
	if idx < 1 || idx > 3 {
		log.Errorf("%s AddSpellCharge傳入錯誤索引: %v", logger.LOG_Redis, idx)
		return
	}
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.spellChargesBuffer[(idx - 1)] += value
}

// 設定掉落道具
func (rPlayer *RedisPlayer) SetDrop(idx int, value int32) {
	rPlayer.MutexLock.Lock()
	defer rPlayer.MutexLock.Unlock()
	rPlayer.dropsBuffer[idx] = value
}

// 暫存資料寫入並每X毫秒更新上RedisDB
func (player *RedisPlayer) updatePlayer() {
	ticker := time.NewTicker(time.Duration(dbWriteMinMiliSecs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-player.MyLoopChan.StopChan:
			log.Infof("%s 關閉RedisPlayer更新", logger.LOG_Redis)
			return
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
func GetPlayerDBData(playerID string) (RedisDBPlayer, error) {
	var player RedisDBPlayer
	playerID = "player-" + playerID
	log.Infof("%s Redis playerID: %v", logger.LOG_Redis, playerID)
	val, err := rdb.HGetAll(ctx, playerID).Result()
	if err != nil {
		log.Errorf("GetPlayerDBData錯誤: %v", err)
		return player, fmt.Errorf("GetPlayerDBData錯誤: %v", err)
	}
	if len(val) == 0 { // 找不到資料回傳0值
		return player, nil
	}

	config := &mapstructure.DecoderConfig{ // 使用自定義的Decode Hook
		DecodeHook: decodeHook,
		Result:     &player,
	}
	decoder, newDecoderErr := mapstructure.NewDecoder(config)
	if newDecoderErr != nil {
		return player, fmt.Errorf("mapstructure.NewDecoder錯誤: %v", newDecoderErr)
	}
	decodeErr := decoder.Decode(val)
	if decodeErr != nil {
		return player, fmt.Errorf("RedisDB Plaeyr 反序列化錯誤: %v", decodeErr)
	}
	// log.Infof("%s playerID: %s point: %d heroExp: %d\n", logger.LOG_Redis, player.ID, player.Point, player.HeroExp)
	return player, nil

}
