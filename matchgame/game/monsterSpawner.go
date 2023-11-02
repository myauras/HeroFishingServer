package game

import (
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"strconv"
	"sync"
	"time"

	"github.com/google/martian/v3/log"
)

type ScheduledSpawn struct {
	MonsterIDs []int
	RouteID    int
	IsBoss     bool
}

func NewScheduledSpawn(monsterIDs []int, routeID int, isBoss bool) *ScheduledSpawn {
	return &ScheduledSpawn{
		MonsterIDs: monsterIDs,
		RouteID:    routeID,
		IsBoss:     isBoss,
	}
}

type MonsterScheduler struct {
	BossExist bool // BOSS是否存在場上的標記

	spawnMonsterQueue chan *ScheduledSpawn // 出怪排程
	spawnTimerMap     map[int]int          // <MonsterSpawn表ID,出怪倒數秒數>

	mutex sync.Mutex
}

func NewMonsterScheduler() *MonsterScheduler {
	return &MonsterScheduler{
		spawnMonsterQueue: make(chan *ScheduledSpawn, 100), // 設定合理的緩衝大小
		spawnTimerMap:     make(map[int]int),
	}
}

// Init 初始化
func (ms *MonsterScheduler) Init(mapData *gameJson.MapJsonData) {
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

	// 啟動出怪檢查任務
	go ms.SpawnCheck()
}

// SpawnCheck 檢查那些出怪表ID需要被加入出怪排程中
func (ms *MonsterScheduler) SpawnCheck() {
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
					if ids == nil || len(ids) == 0 {
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
					ms.spawnMonsterQueue <- spawn // 加入排程
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
					ms.spawnMonsterQueue <- spawn // 加入排程
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

// DequeueMonster 從排程中移除出怪
func (ms *MonsterScheduler) DequeueMonster() *ScheduledSpawn {
	select {
	case spawn := <-ms.spawnMonsterQueue:
		if spawn.IsBoss {
			ms.BossExist = true
		}
		return spawn
	default:
		return nil
	}
}
