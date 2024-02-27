package game

import (
	"herofishingGoModule/gameJson"
	"herofishingGoModule/utility"
	"matchgame/logger"
	"matchgame/packet"
	"math"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Spawn struct {
	SpawnJsonID    int   // 生怪JsonID
	MonsterJsonIDs []int // 怪物JsonID清單
	MonsterIdxs    []int // 怪物唯一索引清單
	RouteJsonID    int   // 路徑JsonID
	IsBoss         bool  // 此生怪是否為BOSS生怪
}

var spawnAccumulator = utility.NewAccumulator() // 產生一個生怪累加器

func NewSpawn(spawnID int, monsterJsonIDs []int, routeJsonID int, isBoss bool) *Spawn {
	// log.Infof("%s 加入生怪駐列 怪物IDs: %v", logger.LOG_MonsterSpawner, monsterIDs)
	monsterIdxs := make([]int, len(monsterJsonIDs))
	return &Spawn{
		SpawnJsonID:    spawnID,
		MonsterJsonIDs: monsterJsonIDs,
		MonsterIdxs:    monsterIdxs,
		RouteJsonID:    routeJsonID,
		IsBoss:         isBoss,
	}
}

type MonsterSpawner struct {
	BossExist     bool             // BOSS是否存在場上的標記
	spawnTimerMap map[int]int      // <MonsterSpawn表ID,出怪倒數秒數>
	Monsters      map[int]*Monster // 目前場上存活的怪物列表
	Spawns        []packet.Spawn   // 生怪清單(如果該生怪中的怪物都死光就會從此清單中移除), 玩家剛加入遊戲時 與 定時同步場景用
	controlChan   chan bool        // 生怪開關Chan
	MutexLock     sync.Mutex
}

func NewMonsterSpawner() *MonsterSpawner {
	return &MonsterSpawner{
		spawnTimerMap: make(map[int]int),
		Monsters:      make(map[int]*Monster),
		Spawns:        make([]packet.Spawn, 0),
		controlChan:   make(chan bool, 1),
	}
}

// 初始化生怪器
func (ms *MonsterSpawner) InitMonsterSpawner(mapJsonID int32) {
	log.Infof("%s 初始化生怪器", logger.LOG_MonsterSpawner)
	mapData, err := gameJson.GetMapByID(strconv.Itoa(int(mapJsonID)))
	if err != nil {
		log.Errorf("%s gameJson.GetMapByID(strconv.Itoa(int(mapJsonID)))錯誤: %v", logger.LOG_MonsterSpawner, err)
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
	log.Infof("%s 生怪器初始化完成", logger.LOG_MonsterSpawner)
}

// 生怪開關控制
func (ms *MonsterSpawner) SpawnSwitch(setOn bool) {
	ms.controlChan <- setOn
	if setOn {
		log.Infof("%s 開始生怪", logger.LOG_MonsterSpawner)
	} else {
		log.Infof("%s 停止生怪", logger.LOG_MonsterSpawner)
	}
}

// 生怪計時器, 執行生怪倒數, Spawner倒數結束就生怪
func (ms *MonsterSpawner) SpawnTimer() {

	running := false
	for {

		select {
		case isOn := <-ms.controlChan:
			running = isOn
		default:
			time.Sleep(time.Second) // 每秒檢查一次
			// 怪物計時器是否正在執行(當房間中沒有玩家時running會是false)
			if !running {
				continue
			}
			// 冰凍檢查, 如果還在冰凍就不產怪
			if MyRoom.OnEffect("Frozen") {
				continue
			}

			// 怪物移除檢查
			needRemoveMonsterIdxs := make([]int, 0)
			for _, monster := range ms.Monsters {
				if MyRoom.GameTime > monster.LeaveTime {
					// log.Errorf("怪物離開: %v", monster.MonsterIdx)
					needRemoveMonsterIdxs = append(needRemoveMonsterIdxs, monster.MonsterIdx)
				}
			}
			if len(needRemoveMonsterIdxs) != 0 {
				ms.RemoveMonsters(needRemoveMonsterIdxs)
			}
			// 生怪檢查
			for spawnID, timer := range ms.spawnTimerMap {
				spawnData, _ := gameJson.GetMonsterSpawnerByID(strconv.Itoa(spawnID)) // 這邊不用檢查err因為會加入spawnTimerMap都是檢查過的
				if ms.BossExist && spawnData.SpawnType == gameJson.Boss {
					continue // BOSS還活著就不會加入BOSS類型的出怪表ID
				}
				timer -= 1
				ms.spawnTimerMap[spawnID] = timer

				if timer <= 0 {
					var spawn *Spawn
					switch spawnData.SpawnType {
					case gameJson.RandomGroup:
						ids, err := utility.Split_INT(spawnData.TypeValue, ",")
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
						monsterJsonIDs, err := newSpawnData.GetMonsterJsonIDs()
						if err != nil {
							log.Errorf("%s newSpawnData.GetMonsterIDs()錯誤: %v", logger.LOG_MonsterSpawner, err)
						}
						routJsonID, err := newSpawnData.GetRandRoutJsonID()
						if err != nil {
							log.Errorf("%s newSpawnData.GetRandRoutID()錯誤: %v", logger.LOG_MonsterSpawner, err)
							continue
						}
						spawn = NewSpawn(rndSpawnID, monsterJsonIDs, routJsonID, newSpawnData.SpawnType == gameJson.Boss)
						ms.Spawn(spawn)
					case gameJson.Minion, gameJson.Boss:
						monsterJsonIDs, err := spawnData.GetMonsterJsonIDs()
						if err != nil {
							log.Errorf("%s spawnData.GetMonsterIDs()錯誤: %v", logger.LOG_MonsterSpawner, err)
						}
						routJsonID, err := spawnData.GetRandRoutJsonID()
						if err != nil {
							log.Errorf("%s spawnData.GetRandRoutID()錯誤: %v", logger.LOG_MonsterSpawner, err)
							continue
						}
						spawn = NewSpawn(spawnID, monsterJsonIDs, routJsonID, spawnData.SpawnType == gameJson.Boss)
						ms.Spawn(spawn)
					}
					spawnSecs, err := spawnData.GetRandSpawnSec()
					if err != nil {
						log.Errorf("%s spawnData.GetRandSpawnSec()錯誤: %v", logger.LOG_MonsterSpawner, err)
					}
					ms.spawnTimerMap[spawnID] = spawnSecs
				}
			}
		}
	}
}

// 生怪並把怪物加入怪物清單 並 廣播給所有玩家
func (ms *MonsterSpawner) Spawn(spawn *Spawn) {
	// log.Infof("%s 生怪IDs: %v", logger.LOG_MonsterSpawner, spawn.MonsterIDs)
	routeJson, err := gameJson.GetRouteByID(strconv.Itoa(spawn.RouteJsonID))
	if err != nil {
		log.Errorf("%s gameJson.GetRouteByID(strconv.Itoa(spawn.RouteJsonID))錯誤: %v", logger.LOG_MonsterSpawner, err)
		return
	}
	routeJsonID, err := strconv.ParseInt(routeJson.ID, 10, 64)
	if err != nil {
		log.Errorf("%s strconv.ParseInt(routeJson.ID, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
		return
	}
	monsters := make([]*packet.Monster, 0)

	// 如果是Boss生怪就將BOSS已存在設定為true
	if spawn.IsBoss {
		ms.MutexLock.Lock()
		ms.BossExist = true
		ms.MutexLock.Unlock()
		// log.Warn("設定BOSS出場")
	}
	// 遍歷生怪中的怪物
	for i, monsterID := range spawn.MonsterJsonIDs {
		monsterJson, err := gameJson.GetMonsterByID(strconv.Itoa(monsterID))
		if err != nil {
			log.Errorf("%s gameJson.GetMonsterByID: %v", logger.LOG_MonsterSpawner, monsterID)
			continue
		}
		monsterJsonID, err := strconv.ParseInt(monsterJson.ID, 10, 64)
		if err != nil {
			log.Errorf("%s strconv.ParseInt(monsterJson.ID, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, monsterID)
			continue
		}

		// 設定怪物唯一索引
		monsterIdx := spawnAccumulator.GetNextIdx("monster")
		// log.Warnf("生怪 MonsterIdx: %v", monsterIdx)
		fromPos, err := utility.NewVector2XZ(routeJson.SpawnPos)
		if err != nil {
			log.Errorf("%s utility.NewVector2XZ(routeJson.SpawnPos)錯誤: %v", logger.LOG_MonsterSpawner, err)
		}
		toPos, err := utility.NewVector2XZ(routeJson.TargetPos)
		if err != nil {
			log.Errorf("%s utility.NewVector2XZ(routeJson.TargetPos)錯誤: %v", logger.LOG_MonsterSpawner, err)
		}
		dist := utility.GetDistance(toPos, fromPos)
		moveSpeed, err := strconv.ParseFloat(monsterJson.Speed, 64)
		if err != nil {
			log.Errorf("%s strconv.ParseFloat(monsterJson.Speed, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
		}
		toTargetTime := dist / moveSpeed
		spawn.MonsterIdxs[i] = monsterIdx
		// 加入怪物清單
		leaveTime := MyRoom.GameTime + toTargetTime
		// if spawn.IsBoss {
		// log.Warnf("monsterIdx:%v GameTime: %v routeID: %s fromPos: %v toPos: %v dist: %v toTargetSec: %v leaveSec: %v", monsterIdx, MyRoom.GameTime, routeJson.ID, fromPos, toPos, dist, toTargetTime, leaveTime)
		// }
		// log.Warnf("monsterIdx:%v GameTime: %v routeID: %s fromPos: %v toPos: %v dist: %v toTargetTime: %v leaveTime: %v", monsterIdx, MyRoom.GameTime, routeJson.ID, fromPos, toPos, dist, toTargetTime, leaveTime)
		ms.Monsters[monsterIdx] = &Monster{
			MonsterJson: monsterJson,
			MonsterIdx:  monsterIdx,
			RouteJson:   routeJson,
			SpawnTime:   MyRoom.GameTime,
			LeaveTime:   leaveTime,
		}

		// 紀錄怪物清單
		monsters = append(monsters, &packet.Monster{
			ID:      int(monsterJsonID),
			Idx:     monsterIdx,
			Death:   false,
			LTime:   math.Round(leaveTime),
			Effects: nil,
		})

	}
	// 紀錄生怪清單
	ms.Spawns = append(ms.Spawns, packet.Spawn{
		RID:   int(routeJsonID),
		STime: MyRoom.GameTime,
		IsB:   spawn.IsBoss,
		Ms:    monsters,
	})
	// 廣播給所有玩家
	MyRoom.BroadCastPacket(-1, &packet.Pack{
		CMD: packet.SPAWN_TOCLIENT,
		Content: &packet.Spawn_ToClient{
			IsBoss:      spawn.IsBoss,
			MonsterIDs:  spawn.MonsterJsonIDs,
			MonsterIdxs: spawn.MonsterIdxs,
			RouteID:     spawn.RouteJsonID,
			SpawnTime:   MyRoom.GameTime,
		},
	})
}

// 從怪物清單中移除被擊殺的怪物
// 從Spawn清單中把死亡的Death設定為true
// 如果某個Spawn的怪物清單都死亡就移除該Spawn
func (ms *MonsterSpawner) RemoveMonsters(killMonsterIdxs []int) {
	if len(killMonsterIdxs) == 0 {
		return
	}
	ms.SendDieMonsters(killMonsterIdxs) // 怪物死亡時廣播封包給client

	killSet := make(map[int]bool)
	for _, v := range killMonsterIdxs {
		killSet[v] = true
	}

	// 檢查Spawn清單是否有Spawn沒有存活的怪物了, 沒有就移除該Spawn事件
	needRemoveSpawnIdxs := make([]int, 0)
	for i, spawn := range ms.Spawns {
		if spawn.Ms == nil {
			needRemoveSpawnIdxs = append(needRemoveSpawnIdxs, i)
			continue
		}

		noAliveMonster := true // 此Spawn是否沒有怪物存活了
		for _, monster := range spawn.Ms {
			if _, exists := killSet[monster.Idx]; exists {
				monster.Death = true // 設定為已死亡
			}
			if !monster.Death {
				noAliveMonster = false
			}
		}
		// 如果此Spawn沒有任何怪物存活就把此Spawn加到要移除清單中
		if noAliveMonster {
			needRemoveSpawnIdxs = append(needRemoveSpawnIdxs, i)
			if spawn.IsB {
				ms.MutexLock.Lock()
				ms.BossExist = false
				ms.MutexLock.Unlock()
				// log.Errorf("設定BOSS退場 spawnIdx: %v", i)
			}
		}

	}
	// log.Warnf("移除 MonsterIdx: %v", killMonsterIdxs)
	utility.RemoveFromMapByKeys(ms.Monsters, killMonsterIdxs) // 從怪物清單中移除被擊殺的怪物
	if len(needRemoveSpawnIdxs) > 0 {                         // 如果有Spawn的怪物都死亡就移除該Spawn
		// log.Infof("%s spawn中沒有怪物存活, 移除該spawn", logger.LOG_MonsterSpawner)
		ms.Spawns = utility.RemoveFromSliceBySlice(ms.Spawns, needRemoveSpawnIdxs)
	}

}

// 怪物死亡時廣播封包給client
func (ms *MonsterSpawner) SendDieMonsters(killMonsterIdxs []int) {
	dieMonsters := make([]packet.DieMonster, 0)

	for _, idx := range killMonsterIdxs {

		for _, monster := range ms.Monsters {
			if monster.MonsterIdx == idx {
				jsonID, err := strconv.ParseInt(monster.MonsterJson.ID, 10, 64)
				if err != nil {
					log.Errorf("%s strconv.ParseInt(monster.MonsterJson.ID, 10, 64)錯誤: %v", logger.LOG_MonsterSpawner, err)
					jsonID = -1
				}
				dieMonsters = append(dieMonsters, packet.DieMonster{
					ID:  int(jsonID),
					Idx: idx,
				})
				break
			}
		}
	}
	if len(dieMonsters) == 0 {
		return
	}

	MyRoom.BroadCastPacket(-1, &packet.Pack{ // 廣播封包
		CMD: packet.MONSTERDIE_TOCLIENT,
		Content: &packet.MonsterDie_ToClient{
			DieMonsters: dieMonsters,
		},
	})
}
