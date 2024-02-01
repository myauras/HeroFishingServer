package game

import (
	"fmt"
	"herofishingGoModule/gameJson"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/utility"
	logger "matchgame/logger"
	"matchgame/packet"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// 處理收到的攻擊事件
func (room *Room) HandleAttack(player *Player, pack packet.Pack, content packet.Attack) {

	// 攻擊ID格式為 [玩家index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	attackID := strconv.Itoa(player.Index) + "_" + strconv.Itoa(content.AttackID)
	if event, ok := room.AttackEvents[attackID]; ok {
		if room.GameTime > event.ExpiredTime { // 此攻擊已經過期
			log.Errorf("%s AttackID: %s 已過期", logger.LOG_Room, attackID)
			return
		}
	}
	// 如果有鎖定目標怪物, 檢查目標怪是否存在, 不存在就返回
	if content.MonsterIdx >= 0 {
		if monster, ok := room.MSpawner.Monsters[content.MonsterIdx]; ok {
			if monster == nil {
				return
			}
		} else {
			return
		}
	}
	// needPoint := int64(room.DBmap.Bet)
	// 取技能表
	spellJson, err := gameJson.GetHeroSpellByID(content.SpellJsonID)
	if err != nil {
		log.Errorf("%s gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
		return
	}
	// 取rtp
	rtp := spellJson.RTP
	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
	spellIdx := 0             // 釋放第幾個技能, 0就代表是普攻
	spendSpellCharge := 0     // 花費技能充能
	spendPoint := int64(0)    // 花費點數

	// 如果是技能攻擊, 設定spellIdx(第幾招技能), 並檢查充能是否足夠
	if isSpellAttack {
		idx, err := utility.ExtractLastDigit(spellJson.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
		spellIdx = idx
		if err != nil {
			room.SendPacketToPlayer(player.Index, &packet.Pack{
				CMD:     packet.HIT_TOCLIENT,
				PackID:  pack.PackID,
				ErrMsg:  "HandleAttack時取技能索引ID錯誤",
				Content: &packet.Hit_ToClient{},
			})
			log.Errorf("%s 取施法技能索引錯誤: %v", logger.LOG_Room, err)
			return
		}
		// 檢查CD
		if spellIdx < 1 || spellIdx > 3 {
			log.Errorf("%s 技能索引不為1~3: %v", logger.LOG_Room, spellIdx)
			return
		}
		passSec := room.GameTime - player.LastSpellsTime[spellIdx-1] // 距離上次攻擊經過的秒數
		if passSec < spellJson.CD {
			log.Errorf("%s 玩家%s的技能仍在CD中, 不應該能施放技能, passSec: %v cd: %v", logger.LOG_Room, player.DBPlayer.ID, passSec, spellJson.CD)
			return
		}
		// 檢查是否可以施放該技能
		if player.CanSpell(spellIdx) {
			log.Errorf("%s 該玩家充能不足, 無法使用技能才對", logger.LOG_Room)
			return
		}
		spell, getSpellErr := player.MyHero.GetSpell(spellIdx)
		if getSpellErr != nil {
			log.Errorf("%s player.MyHero.GetSpell(spellIdx)錯誤: %v", logger.LOG_Room, getSpellErr)
			return
		}

		spendSpellCharge = spell.Cost
		player.LastSpellsTime[spellIdx-1] = room.GameTime

	} else { // 如果是普攻, 檢查是否有足夠點數
		// 檢查CD, 普攻的CD要考慮Buff
		// passSec := room.GameTime - player.LastAttackTime // 距離上次攻擊經過的秒數
		// cd := spellJson.CD / player.GetAttackCDBuff()    // CD秒數
		// if passSec < cd {
		// 	log.Errorf("%s 玩家%s的攻擊仍在CD中, 不應該能攻擊, passSec: %v cd: %v", logger.LOG_Room, player.DBPlayer.ID, passSec, cd)
		// 	return
		// }

		// (先關閉點數不足檢測)
		// 檢查點數
		// if player.DBPlayer.Point < needPoint {
		// 	log.Errorf("%s 該玩家點數不足, 無法普攻才對", logger.LOG_Room)
		// 	return
		// }
		spendPoint = -int64(room.DBmap.Bet)
		player.LastAttackTime = room.GameTime // 設定上一次攻擊時間
	}
	// =============建立攻擊事件=============
	var attackEvent *AttackEvent
	// 以attackID來建立攻擊事件, 如果攻擊事件已存在代表是同一個技能但不同波次的攻擊, 此時就追加擊中怪物清單在該攻擊事件
	if _, ok := room.AttackEvents[attackID]; !ok {
		idxs := make([][]int, 0)
		attackEvent = &AttackEvent{
			AttackID:          attackID,
			ExpiredTime:       room.GameTime + ATTACK_EXPIRED_SECS,
			MonsterIdxs:       idxs,
			Paid:              true,
			Hit_ToClientPacks: make([]packet.Pack, 0),
		}
		room.AttackEvents[attackID] = attackEvent // 將此攻擊事件加入清單
	} else { // 有同樣的攻擊事件存在代表Hit比Attack先送到
		attackEvent = room.AttackEvents[attackID]
		attackEvent.Paid = true // 設為已支付費用
		// 有Hit先送到的封包要處理
		if len(attackEvent.Hit_ToClientPacks) > 0 {
			for _, v := range attackEvent.Hit_ToClientPacks {
				room.settleHit(player, v)
			}
		}
	}

	// =============是合法的攻擊就進行資源消耗與回送封包=============

	// 玩家點數變化
	player.AddPoint(spendPoint)
	// 施放技能的話要減少英雄技能充能
	if spellIdx != 0 && spendSpellCharge != 0 {
		player.AddSpellCharge(spellIdx, -spendSpellCharge)
	}

	// 廣播給client
	room.BroadCastPacket(player.Index, &packet.Pack{
		CMD:    packet.ATTACK_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.Attack_ToClient{
			PlayerIdx:   player.Index,
			SpellJsonID: content.SpellJsonID,
			MonsterIdx:  content.MonsterIdx,
			AttackLock:  content.AttackLock,
			AttackPos:   content.AttackPos,
			AttackDir:   content.AttackDir,
		}},
	)
}

// 處理收到的擊中事件
func (room *Room) HandleHit(player *Player, pack packet.Pack, content packet.Hit) {
	// 攻擊ID格式為 [玩家index]_[攻擊流水號] (攻擊流水號(AttackID)是client端送來的施放攻擊的累加流水號
	// EX. 2_3就代表房間座位2的玩家進行的第3次攻擊
	attackID := strconv.Itoa(player.Index) + "_" + strconv.Itoa(content.AttackID)
	if event, ok := room.AttackEvents[attackID]; ok {
		if room.GameTime > event.ExpiredTime { // 此攻擊已經過期
			log.Errorf("%s AttackID: %s 已過期", logger.LOG_Room, attackID)
			return
		}
	}

	// 取技能表
	spellJson, err := gameJson.GetHeroSpellByID(content.SpellJsonID)
	if err != nil {
		room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤", pack))
		log.Errorf("%s HandleHit時gameJson.GetHeroSpellByID(hitCMD.SpellJsonID)錯誤: %v", logger.LOG_Room, err)
		return
	}
	// 取rtp
	rtp := spellJson.RTP
	isSpellAttack := rtp != 0 // 此攻擊的spell表的RTP不是0就代表是技能攻擊
	// 取波次命中數
	spellMaxHits := spellJson.MaxHits

	// hitMonsterIdxs := make([]int, 0)   // 擊中怪物索引清單
	killMonsterIdxs := make([]int, 0)  // 擊殺怪物索引清單, [1,1,3]就是依次擊殺索引為1,1與3的怪物
	gainPoints := make([]int64, 0)     // 獲得點數清單, [1,1,3]就是依次獲得點數1,1與3
	gainSpellCharges := make([]int, 0) // 獲得技能充能清單, [1,1,3]就是依次獲得技能1,技能1,技能3的充能
	gainHeroExps := make([]int, 0)     // 獲得英雄經驗清單, [1,1,3]就是依次獲得英雄經驗1,1與3
	gainDrops := make([]int, 0)        // 獲得掉落清單, [1,1,3]就是依次獲得DropJson中ID為1,1與3的掉落
	// 遍歷擊中的怪物並計算擊殺與獎勵
	content.MonsterIdxs = utility.RemoveDuplicatesFromSlice(content.MonsterIdxs) // 移除重複的命中索引
	for _, monsterIdx := range content.MonsterIdxs {
		// 確認怪物索引存在清單中, 不存在代表已死亡或是client送錯怪物索引
		if monster, ok := room.MSpawner.Monsters[monsterIdx]; !ok {
			errStr := fmt.Sprintf("目標不存在(或已死亡) monsterIdx:%d", monsterIdx)
			room.SendPacketToPlayer(player.Index, newHitErrorPack(errStr, pack))
			log.Errorf("%s %s", logger.LOG_Room, errStr)
			continue
		} else {
			if monster == nil {
				room.SendPacketToPlayer(player.Index, newHitErrorPack("room.MSpawner.Monsters中的monster is null", pack))
				log.Errorf("%s room.MSpawner.Monsters中的monster is null", logger.LOG_Room)
				continue
			}

			// hitMonsterIdxs = append(hitMonsterIdxs, monsterIdx) // 加入擊中怪物索引清單

			// 取得怪物賠率
			odds, err := strconv.ParseFloat(monster.MonsterJson.Odds, 64)
			if err != nil {
				room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取怪物賠率錯誤", pack))
				log.Errorf("%s strconv.ParseFloat(monster.MonsterJson.Odds, 64)錯誤: %v", logger.LOG_Room, err)
				return
			}
			// 取得怪物經驗
			monsterExp, err := strconv.ParseFloat(monster.MonsterJson.EXP, 64)
			if err != nil {
				room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取怪物經驗錯誤", pack))
				log.Errorf("%s strconv.ParseFloat(monster.MonsterJson.EXP, 64)錯誤: %v", logger.LOG_Room, err)
				return
			}

			// 取得怪物掉落道具
			dropAddOdds := 0.0   // 掉落道具增加的總RTP
			dropID64 := int64(0) // 怪物掉落ID
			// 怪物必須有掉落物才需要考慮怪物掉落
			if monster.MonsterJson.DropID != "" {
				log.Errorf("monster.MonsterJson.DropID= %s", monster.MonsterJson.DropID)
				dropJson, err := gameJson.GetDropByID(monster.MonsterJson.DropID)
				if err != nil {
					room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取掉落表錯誤", pack))
					log.Errorf("%s HandleHit時gameJson.GetDropByID(monster.MonsterJson.DropID)錯誤: %v", logger.LOG_Room, err)
					return
				}
				dropID64, err = strconv.ParseInt(monster.MonsterJson.DropID, 10, 64)
				if err != nil {
					log.Errorf("%s HandleHit時strconv.ParseInt(monster.MonsterJson.DropID, 10, 64)錯誤: %v", logger.LOG_Room, err)
					return
				}
				// 玩家目前還沒擁有該掉落ID 才需要考慮怪物掉落
				if !player.IsOwnedDrop(int(dropID64)) {
					addOdds, err := strconv.ParseFloat(dropJson.RTP, 64)
					if err != nil {
						room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時取掉落表的賠率錯誤", pack))
						log.Errorf("%s HandleHit時strconv.ParseFloat(dropJson.GainRTP, 64)錯誤: %v", logger.LOG_Room, err)
						return
					}
					dropAddOdds += addOdds
				}
			}

			// 計算實際怪物死掉獲得點數
			rewardPoint := int64((odds + dropAddOdds) * float64(room.DBmap.Bet))

			// 計算是否造成擊殺
			kill := false
			rndUnchargedSpell, gotUnchargedSpell := player.GetRandomUnchargedSpell() // 計算是否有尚未充滿能的技能, 有的話隨機取一個未充滿能的技能
			if !isSpellAttack {                                                      // 普攻
				// 擊殺判定
				attackKP := room.MathModel.GetAttackKP(odds, int(spellMaxHits), gotUnchargedSpell)
				kill = utility.GetProbResult(attackKP)
				// log.Infof("======spellMaxHits:%v odds:%v attackKP:%v kill:%v ", spellMaxHits, odds, attackKP, kill)
			} else { // 技能攻擊
				attackKP := room.MathModel.GetSpellKP(rtp, odds, int(spellMaxHits))
				kill = utility.GetProbResult(attackKP)
				// log.Infof("======spellMaxHits:%v rtp: %v odds:%v attackKP:%v kill:%v", spellMaxHits, rtp, odds, attackKP, kill)
			}

			// 如果有擊殺就加到清單中
			if kill {
				// 技能充能掉落
				dropChargeP := 0.0
				gainSpellCharges = append(gainSpellCharges, -1)
				gainDrops = append(gainDrops, -1)
				if gotUnchargedSpell {
					dropChargeP = room.MathModel.GetHeroSpellDropP_AttackKilling(rndUnchargedSpell.RTP, odds)
					if utility.GetProbResult(dropChargeP) {
						dropSpellIdx, err := utility.ExtractLastDigit(rndUnchargedSpell.ID) // 掉落充能的技能索引(1~3) Ex.1就是第1個技能
						if err != nil {
							log.Errorf("%s HandleHit時utility.ExtractLastDigit(rndUnchargedSpell.ID)錯誤: %v", logger.LOG_Room, err)
							room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時解析第X技能索引錯誤", pack))
							return
						}
						gainSpellCharges[len(gainSpellCharges)-1] = dropSpellIdx
					}
				}
				// log.Errorf("擊殺怪物: %v", monsterIdx)
				killMonsterIdxs = append(killMonsterIdxs, monsterIdx)
				gainPoints = append(gainPoints, rewardPoint)
				gainHeroExps = append(gainHeroExps, int(monsterExp))
				if dropID64 != 0 {
					gainDrops[len(gainDrops)-1] = int(dropID64)
				}
			}
		}
	}

	// 設定AttackEvent
	var attackEvent *AttackEvent
	// 不存在此攻擊事件代表之前的Attack封包還沒送到
	if _, ok := room.AttackEvents[attackID]; !ok {
		idxs := make([][]int, 0)
		attackEvent = &AttackEvent{
			AttackID:          attackID,
			ExpiredTime:       room.GameTime + ATTACK_EXPIRED_SECS,
			MonsterIdxs:       idxs,
			Paid:              false, // 設定為還沒支付費用
			Hit_ToClientPacks: make([]packet.Pack, 0),
		}
		room.AttackEvents[attackID] = attackEvent // 將此攻擊事件加入清單

	} else {
		attackEvent = room.AttackEvents[attackID]
		if attackEvent == nil {
			room.SendPacketToPlayer(player.Index, newHitErrorPack("HandleHit時room.AttackEvents[attackID]為nil", pack))
			log.Errorf("%s room.AttackEvents[attackID]為nil", logger.LOG_Room)
			return
		}
	}

	// 計算目前此技能收到的總擊中數量 並檢查 是否超過此技能的最大擊中數量
	hitCount := 0
	for _, innerSlice := range attackEvent.MonsterIdxs {
		hitCount += len(innerSlice)
	}
	if hitCount >= int(spellMaxHits) {
		log.Error(content.MonsterIdxs)
		errLog := fmt.Sprintf("HandleHit時收到的擊中數量超過此技能最大可擊中數量, SpellID: %s curHit: %v MonsterIdxs: %v", spellJson.ID, hitCount, attackEvent.MonsterIdxs)
		log.Error(errLog)
		room.SendPacketToPlayer(player.Index, newHitErrorPack(errLog, pack))

		return
	}
	attackEvent.MonsterIdxs = append(attackEvent.MonsterIdxs, content.MonsterIdxs) // 將此波命中加入攻擊事件中
	// 將命中結果封包計入在此攻擊事件中
	hitPack := packet.Pack{
		CMD:    packet.HIT_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.Hit_ToClient{
			PlayerIdx:        player.Index,
			KillMonsterIdxs:  killMonsterIdxs,
			GainPoints:       gainPoints,
			GainHeroExps:     gainHeroExps,
			GainSpellCharges: gainSpellCharges,
			GainDrops:        gainDrops,
		}}
	attackEvent.Hit_ToClientPacks = append(attackEvent.Hit_ToClientPacks, hitPack)
	// log.Errorf("attackEvent.Paid: %v   killMonsterIdxs: %v", attackEvent.Paid, killMonsterIdxs)
	// =============已完成支付費用的命中就進行資源消耗與回送封包=============
	if attackEvent.Paid {
		room.settleHit(player, hitPack)
	}

}

// 已付費的Attack事件才會結算命中
func (room *Room) settleHit(player *Player, hitPack packet.Pack) {

	var content *packet.Hit_ToClient
	if c, ok := hitPack.Content.(*packet.Hit_ToClient); !ok {
		log.Errorf("%s hitPack.Content無法斷言為Hit_ToClient", logger.LOG_Room)
		return
	} else {
		content = c
	}
	// 玩家點數變化
	totalGainPoint := utility.SliceSum(content.GainPoints) // 把 每個擊殺獲得點數加總就是 總獲得點數
	if totalGainPoint != 0 {
		player.AddPoint(totalGainPoint)
	}

	// 英雄增加經驗
	totalGainHeroExps := utility.SliceSum(content.GainHeroExps) // 把 每個擊殺獲得英雄經驗加總就是 總獲得英雄經驗
	player.AddHeroExp(totalGainHeroExps)
	// 擊殺怪物增加英雄技能充能
	for _, v := range content.GainSpellCharges {
		if v <= 0 { // 因為有擊殺但沒掉落充能時, gainSpellCharges仍會填入-1, 所以要加判斷
			continue
		}
		player.AddSpellCharge(v, 1)
	}
	// 擊殺怪物獲得掉落道具
	for _, dropID := range content.GainDrops {
		if dropID <= 0 { // 因為有擊殺但沒掉落時, gainDrops仍會填入-1, 所以要加判斷
			continue
		}
		player.AddDrop(dropID)
	}
	// 從怪物清單中移除被擊殺的怪物(付費後才算目標死亡, 沒收到付費的Attack封包之前都還是算怪物存活)
	room.MSpawner.RemoveMonsters(content.KillMonsterIdxs)
	log.Infof("killMonsterIdxs: %v gainPoints: %v gainHeroExps: %v gainSpellCharges: %v  , gainDrops: %v ", content.KillMonsterIdxs, content.GainPoints, content.GainHeroExps, content.GainSpellCharges, content.GainDrops)
	// log.Infof("/////////////////////////////////")
	// log.Infof("killMonsterIdxs: %v \n", killMonsterIdxs)
	// log.Infof("gainPoints: %v \n", gainPoints)
	// log.Infof("gainHeroExps: %v \n", gainHeroExps)
	// log.Infof("gainSpellCharges: %v \n", gainSpellCharges)
	// log.Infof("gainDrops: %v \n", gainDrops)
	// 廣播給client
	room.BroadCastPacket(-1, &hitPack)
}

// 處理收到的掉落施法封包(TCP)
func (room *Room) HandleDropSpell(player *Player, pack packet.Pack, content packet.DropSpell) {
	dropSpellJson, err := gameJson.GetDropSpellByID(strconv.Itoa(content.DropSpellJsonID))
	if err != nil {
		log.Errorf("%s HandleDropSpell時gameJson.GetDropSpellByID(strconv.Itoa(content.DropSpellJsonID))錯誤: %v", logger.LOG_Room, err)
		return
	}
	dropSpellID, err := strconv.ParseInt(dropSpellJson.ID, 10, 64)
	if err != nil {
		log.Errorf("%s HandleDropSpell時strconv.ParseInt(dropSpellJson.ID, 10, 64)錯誤: %v", logger.LOG_Room, err)
		return
	}
	ownedDrop := player.IsOwnedDrop(int(dropSpellID))
	if !ownedDrop {
		log.Errorf("%s 玩家%s 無此DropID, 不應該能使用DropSpell: %v", logger.LOG_Room, player.DBPlayer.ID, dropSpellID)
		return
	}
	switch dropSpellJson.EffectType {
	case "Frozen": // 冰風暴
		duration, err := strconv.ParseFloat(dropSpellJson.EffectValue1, 64)
		if err != nil {
			log.Errorf("%s HandleDropSpell的EffectType為%s時 conv.ParseFloat(dropSpellJson.EffectValue1, 64)錯誤: %v", logger.LOG_Room, dropSpellJson.EffectType, err)
			return
		}
		room.SceneEffects = append(room.SceneEffects, packet.SceneEffect{
			Name:     dropSpellJson.EffectType,
			AtTime:   room.GameTime,
			Duration: duration,
		})
		room.BroadCastPacket(player.Index, &packet.Pack{
			CMD:    packet.UPDATESCENE_TOCLIENT,
			PackID: -1,
			Content: &packet.UpdateScene_ToClient{
				Spawns:       room.MSpawner.Spawns,
				SceneEffects: room.SceneEffects,
			},
		})
	case "Speedup": // 急速神符
		duration, err := strconv.ParseFloat(dropSpellJson.EffectValue1, 64)
		if err != nil {
			log.Errorf("%s HandleDropSpell的EffectType為%s時 strconv.ParseFloat(dropSpellJson.EffectValue1, 64)錯誤: %v", logger.LOG_Room, dropSpellJson.EffectType, err)
			return
		}
		value, err := strconv.ParseFloat(dropSpellJson.EffectValue2, 64)
		if err != nil {
			log.Errorf("%s HandleDropSpell的EffectType為%s時 strconv.ParseFloat(dropSpellJson.EffectValue2, 64)錯誤: %v", logger.LOG_Room, dropSpellJson.EffectType, err)
			return
		}
		player.PlayerBuffs = append(player.PlayerBuffs, packet.PlayerBuff{
			Name:     dropSpellJson.EffectType,
			Value:    value,
			AtTime:   room.GameTime,
			Duration: duration,
		})
		room.BroadCastPacket(player.Index, &packet.Pack{
			CMD:    packet.UPDATEPLAYER_TOCLIENT,
			PackID: -1,
			Content: &packet.UpdatePlayer_ToClient{
				Players: room.GetPacketPlayers(),
			},
		})
	default:
		log.Errorf("%s HandleDropSpell傳入尚未定義的EffectType類型: %v", logger.LOG_Room, dropSpellJson.EffectType)
		return
	}
	// 施法後要移除該掉落
	player.RemoveDrop(int(dropSpellID))
}

// 處理收到的自動攻擊封包(TCP)
func (room *Room) HandleAuto(player *Player, pack packet.Pack, content packet.Auto) {
	isAuto := content.IsAuto
	room.SendPacketToPlayer(player.Index, &packet.Pack{
		CMD:    packet.AUTO_TOCLIENT,
		PackID: -1,
		Content: &packet.Auto_ToClient{
			IsAuto: isAuto,
		},
	})
}

// 取得hitError封包
func newHitErrorPack(errStr string, pack packet.Pack) *packet.Pack {
	return &packet.Pack{
		CMD:     packet.HIT_TOCLIENT,
		PackID:  pack.PackID,
		ErrMsg:  errStr,
		Content: &packet.Hit_ToClient{},
	}
}

// 將房間資料寫入DB(只有開房時執行1次)
func (room *Room) WriteMatchgameToDB() {
	log.Infof("%s 開始寫入Matchgame到DB", logger.LOG_Room)
	_, err := mongo.AddDocByStruct(mongo.ColName.Matchgame, room.DBMatchgame)
	if err != nil {
		log.Errorf("%s writeMatchgameToDB: %v", logger.LOG_Room, err)
		return
	}
	log.Infof("%s 寫入Matchgame到DB完成", logger.LOG_Room)
}
