package game

import (
	// log "github.com/sirupsen/logrus"
	"herofishingGoModule/utility"
	"matchgame/packet"

	"github.com/google/martian/log"
)

// 是否處於某場景效果狀態下
func (room *Room) OnEffect(effectType string) bool {
	for _, v := range room.SceneEffects {
		if v.Name != effectType {
			continue
		}
		endTime := v.AtTime + v.Duration
		if room.GameTime < endTime {
			return true
		}
	}
	return false
}

// 移除過期的場景效果
func (r *Room) RemoveExpiredSceneEffects() {

	toRemoveIdxs := make([]int, 0)
	for i, v := range r.SceneEffects {
		if r.GameTime > v.RemoveTime {
			toRemoveIdxs = append(toRemoveIdxs, i)
		}
	}
	if len(toRemoveIdxs) > 0 {
		// for _, v := range toRemoveIdxs {
		// 	log.Infof("%s 移除過期的場景效果: %v", logger.LOG_Room, r.SceneEffects[v].Name)
		// }
		r.MutexLock.Lock()
		r.SceneEffects = utility.RemoveFromSliceBySlice(r.SceneEffects, toRemoveIdxs)
		defer r.MutexLock.Unlock()
	}
}

// 賦予場景冰凍效果
func (room *Room) AddFrozenEffect(effectType string, duration float64) {
	room.MutexLock.Lock()
	defer room.MutexLock.Unlock()

	// 增加怪物存活時間
	lastMonsterLeaveTime := room.GameTime + duration
	for _, monster := range room.MSpawner.Monsters {
		monster.LeaveTime += duration
		if monster.LeaveTime > lastMonsterLeaveTime {
			lastMonsterLeaveTime = monster.LeaveTime
		}
	}
	// 本來就存在的冰凍效果移除時間也要追加
	for i := range room.SceneEffects {
		room.SceneEffects[i].RemoveTime += duration
	}
	// 加入冰凍效果
	room.SceneEffects = append(room.SceneEffects, packet.SceneEffect{
		Name:       effectType,
		AtTime:     room.GameTime,
		Duration:   duration,
		RemoveTime: lastMonsterLeaveTime,
	})
	log.Errorf("冰凍 開始: %v 結束: %v 移除: %v", room.GameTime, room.GameTime+duration, lastMonsterLeaveTime)
}
