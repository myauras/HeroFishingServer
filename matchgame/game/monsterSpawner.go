package game

import (
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type ScheduledSpawn struct {
	MonsterIDs []int
	RouteID    int
	IsBoss     bool
}
type Monster struct {
	MonsterJson gameJson.MonsterJsonData // 怪物表Json
	RouteJson   gameJson.RouteJsonData   // 路徑表Json
	SpawnTime   float64                  // 在遊戲時間第X秒時被產生的
}

func NewScheduledSpawn(monsterIDs []int, routeID int, isBoss bool) *ScheduledSpawn {
	log.Infof("%s 加入生怪駐列 怪物IDs: %v", logger.LOG_MonsterSpawner, monsterIDs)
	return &ScheduledSpawn{
		MonsterIDs: monsterIDs,
		RouteID:    routeID,
		IsBoss:     isBoss,
	}
}

var MyMonsterScheduler MonsterSpawner // 怪物產生器

type MonsterSpawner struct {
	BossExist     bool               // BOSS是否存在場上的標記
	spawnTimerMap map[int]int        // <MonsterSpawn表ID,出怪倒數秒數>
	Monsters      map[int64]*Monster // 目前場上的怪物列表
	mutex         sync.Mutex
}

func NewMonsterScheduler() *MonsterSpawner {
	return &MonsterSpawner{
		spawnTimerMap: make(map[int]int),
	}
}

// 初始化生怪器
func (ms *MonsterSpawner) InitMonsterSpawner(mapID int32) {
	log.Infof("%s 初始化生怪器", logger.LOG_MonsterSpawner)
	mapData, err := gameJson.GetMapByID(strconv.Itoa(int(mapID)))
	if err != nil {
		log.Errorf("%s gameJson.GetMapByID(strconv.Itoa(mapID))錯誤: %v", logger.LOG_MonsterSpawner, err)
		return
	}

	ms.spawnTimerMap = make(map[int]int)
	mosnterIDs, err := mapData.GetMonsterSpawnerIDs()
	if err != nil {
		log.Errorf("%s mapData.GetMonsterSpawnerIDs()錯誤: %v", logger.LOG_MonsterSpawner, err)
		return
	}
	for _, id := range mosnterIDs {
		spawnData, err := gameJson.GetMonsterSpawnerByID(strconv.Itoa(id))
		if err != nil {
			continue
		}
		spawnSecs, err := spawnData.GetRandSpawnSec()
		if err != nil {
			log.Errorf("%s spawnData.GetRandSpawnSec()錯誤: %v", logger.LOG_MonsterSpawner, err)
		}
		ms.spawnTimerMap[id] = spawnSecs
	}
	log.Infof("%s 生怪器初始化完成, 開始跑生怪循環", logger.LOG_MonsterSpawner)
	// 開始生怪計時器
	go ms.ScheduleMonster()
}

// 生怪計時器, 執行生怪倒數, Spawner倒數結束就生怪
func (ms *MonsterSpawner) ScheduleMonster() {
	for {

		time.Sleep(1000 * time.Millisecond) // 每秒檢查一次
		for spawnID, timer := range ms.spawnTimerMap {
			spawnData, _ := gameJson.GetMonsterSpawnerByID(strconv.Itoa(spawnID)) // 這邊不用檢查err因為會加入spawnTimerMap都是檢查過的
			if ms.BossExist && spawnData.SpawnType == gameJson.Boss {
				continue // BOSS還活著就不會加入BOSS類型的出怪表ID
			}
			timer -= 1
			ms.mutex.Lock()
			ms.spawnTimerMap[spawnID] = timer
			ms.mutex.Unlock()

			if timer <= 0 {
				var spawn *ScheduledSpawn
				switch spawnData.SpawnType {
				case gameJson.RandomGroup:

					ids, err := utility.StrToIntSlice(spawnData.TypeValue, ",")
					if err != nil {
						log.Errorf("%s spawnData ID為 %s 的TypeValue不是,分割的字串: %v", logger.LOG_MonsterSpawner, spawnData.ID, err)
						continue
					}
					if len(ids) == 0 {
						log.Errorf("%s spawnData ID為 %s 的TypeValue填表錯誤: %v", logger.LOG_MonsterSpawner, spawnData.ID, err)
						continue
					}
					rndSpawnID, err := utility.GetRandomTFromSlice(ids)
					if err != nil {
						continue
					}
					newSpawnData, _ := gameJson.GetMonsterSpawnerByID(strconv.Itoa(rndSpawnID))
					monsterIDs, err := newSpawnData.GetMonsterIDs()
					if err != nil {
						log.Errorf("%s newSpawnData.GetMonsterIDs()錯誤: %v", logger.LOG_MonsterSpawner, err)
					}
					routID, err := newSpawnData.GetRandRoutID()
					if err != nil {
						log.Errorf("%s newSpawnData.GetRandRoutID()錯誤: %v", logger.LOG_MonsterSpawner, err)
						continue
					}
					spawn = NewScheduledSpawn(monsterIDs, routID, newSpawnData.SpawnType == gameJson.Boss)
					ms.Spawn(spawn)
				case gameJson.Minion, gameJson.Boss:
					monsterIDs, err := spawnData.GetMonsterIDs()
					if err != nil {
						log.Errorf("%s spawnData.GetMonsterIDs()錯誤: %v", logger.LOG_MonsterSpawner, err)
					}
					routID, err := spawnData.GetRandRoutID()
					if err != nil {
						log.Errorf("%s spawnData.GetRandRoutID()錯誤: %v", logger.LOG_MonsterSpawner, err)
						continue
					}
					spawn = NewScheduledSpawn(monsterIDs, routID, spawnData.SpawnType == gameJson.Boss)
					ms.Spawn(spawn)
				}
				ms.mutex.Lock()
				spawnSecs, err := spawnData.GetRandSpawnSec()
				if err != nil {
					log.Errorf("%s spawnData.GetRandSpawnSec()錯誤: %v", logger.LOG_MonsterSpawner, err)
				}
				ms.spawnTimerMap[spawnID] = spawnSecs
				ms.mutex.Unlock()
			}
		}
	}
}

// 生怪並把怪物加入怪物清單 並 廣播給所有玩家
func (ms *MonsterSpawner) Spawn(spawn *ScheduledSpawn) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	log.Infof("%s 生怪IDs: %v", logger.LOG_MonsterSpawner, spawn.MonsterIDs)
	for monsterID := range spawn.MonsterIDs {
		log.Infof("%s 生怪ID: %v", logger.LOG_MonsterSpawner, monsterID)
		monsterJson, err := gameJson.GetMonsterByID(strconv.Itoa(monsterID))
		if err != nil {
			log.Errorf("%s gameJson.GetMonsterByID: %v", logger.LOG_MonsterSpawner, monsterID)
			continue
		}
		monsterIdx := utility.Accumulator.GetNextIndex("monster", 1)
		routeJson, err := gameJson.GetRouteByID(strconv.Itoa(spawn.RouteID))
		if err != nil {
			log.Errorf("%s gameJson.GetRouteByID: %v", logger.LOG_MonsterSpawner, spawn.RouteID)
			continue
		}

		// 加入怪物清單
		ms.Monsters[monsterIdx] = &Monster{
			MonsterJson: monsterJson,
			RouteJson:   routeJson,
			SpawnTime:   MyRoom.GameTime,
		}

		// 廣播給所有玩家
		MyRoom.broadCastPacket(&packet.Pack{
			CMD: packet.SPAWNM,
			Content: &packet.SpawnCMD{
				MonsterID: monsterID,
				RouteID:   spawn.RouteID,
				SpawnTime: MyRoom.GameTime,
			},
		})
	}
}
