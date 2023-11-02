package game

import (
	"github.com/google/martian/v3/log"
	"herofishingGoModule/gameJson"
	"matchgame/logger"
	logger "matchgame/logger"
	"strconv"
	"sync"
	"time"
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
	spawnTimerMap     map[int]float64      // <MonsterSpawn表ID,出怪倒數秒數>

	mutex sync.Mutex
}

func NewMonsterScheduler() *MonsterScheduler {
	return &MonsterScheduler{
		spawnMonsterQueue: make(chan *ScheduledSpawn, 100), // 設定合理的緩衝大小
		spawnTimerMap:     make(map[int]float64),
	}
}

// Init 初始化
func (ms *MonsterScheduler) Init(mapData *gameJson.MapJsonData) {
	ms.spawnTimerMap = make(map[int]float64)
	mosnterIDs, err := mapData.GetMonsterSpawnerIDs()
	if err != nil {
		log.Errorf("%s mapData.GetMonsterSpawnerIDs錯誤: %v", logger.LOG_MonsterSpawner, err)
		return
	}
	for _, id := range mosnterIDs {
		spawnData, err := gameJson.GetMonsterSpawnerByID(strconv.Itoa(id))
		if err != nil {
			continue
		}
		spawnSecs, err := spawnData.GetRandSpawnSec()
		if err != nil {
			log.Errorf("%s spawnData.GetRandSpawnSec錯誤: %v", logger.LOG_MonsterSpawner, err)
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
		for id, timer := range ms.spawnTimerMap {
			spawnData := GetMonsterSpawnerJsonData(id)
			if spawnData == nil {
				continue
			}
			if ms.BossExist && spawnData.MySpawnType == Boss {
				continue // BOSS還活著就不會加入BOSS類型的出怪表ID
			}
			timer--
			ms.mutex.Lock()
			ms.spawnTimerMap[id] = timer
			ms.mutex.Unlock()

			if timer <= 0 {
				var spawn *ScheduledSpawn
				switch spawnData.MySpawnType {
				case RandomGroup:
					ids := StringSplitToIntArray(spawnData.TypeValue, ',')
					if ids == nil || len(ids) == 0 {
						continue
					}
					newSpawnID := GetRandomFromIntArray(ids)
					newSpawnData := GetMonsterSpawnerJsonData(newSpawnID)
					if newSpawnData == nil {
						continue
					}
					spawn = NewScheduledSpawn(newSpawnData.MonsterIDs, newSpawnData.GetRandRoute(), newSpawnData.MySpawnType == Boss)
					ms.spawnMonsterQueue <- spawn // 加入排程
				case Minion, Boss:
					spawn = NewScheduledSpawn(spawnData.MonsterIDs, spawnData.GetRandRoute(), spawnData.MySpawnType == Boss)
					ms.spawnMonsterQueue <- spawn // 加入排程
				}
				ms.mutex.Lock()
				ms.spawnTimerMap[id] = spawnData.GetRandSpawnSec()
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
