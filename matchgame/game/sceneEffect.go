package game

import (
	// log "github.com/sirupsen/logrus"
	"herofishingGoModule/utility"
	"matchgame/packet"
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
		endTime := v.AtTime + v.Duration
		if r.GameTime > endTime {
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
	// room.MutexLock.Lock()
	// defer room.MutexLock.Unlock()
	room.SceneEffects = append(room.SceneEffects, packet.SceneEffect{
		Name:     effectType,
		AtTime:   room.GameTime,
		Duration: duration,
	})
}
