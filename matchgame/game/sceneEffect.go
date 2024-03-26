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
		if room.GameTime < v.EndTime {
			return true
		}
	}
	return false
}

// 移除過期的場景效果
func (r *Room) RemoveExpiredSceneEffects() {

	toRemoveIdxs := make([]int, 0)
	for i, v := range r.SceneEffects {
		if r.GameTime > v.EndTime {
			toRemoveIdxs = append(toRemoveIdxs, i)
		}
	}
	if len(toRemoveIdxs) > 0 {
		// for _, v := range toRemoveIdxs {
		// 	log.Infof("%s 移除過期的場景效果: %v", logger.LOG_Room, r.SceneEffects[v].Name)
		// }
		r.MutexLock.Lock()
		r.SceneEffects = utility.RemoveFromSliceByIdxs(r.SceneEffects, toRemoveIdxs)
		r.MutexLock.Unlock()
	}
}

// 賦予場景冰凍效果
func (room *Room) AddSceneEffect(effectType string, duration float64) {
	room.MutexLock.Lock()
	defer room.MutexLock.Unlock()

	// 場景加入效果
	room.SceneEffects = append(room.SceneEffects, packet.SceneEffect{
		Name:    effectType,
		AtTime:  room.GameTime,
		EndTime: room.GameTime + duration,
	})

	// 根據場景效果做不同處理
	switch effectType {
	case "Forzen":
		// 場上怪物都賦予冰凍效果
		for _, m := range room.MSpawner.Monsters {
			mEffect := MonsterEffect{
				Name:    effectType,
				AtTime:  MyRoom.GameTime,
				EndTime: MyRoom.GameTime + duration,
			}
			m.AddEffect(mEffect)
		}
	}

}
