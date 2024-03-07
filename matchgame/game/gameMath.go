package game

import (
	"matchgame/logger"
	"math"

	log "github.com/sirupsen/logrus"
)

// 模型結構
type MathModel struct {
	GameRTP                 float64
	SpellSharedRTP          float64
	RtpAdjust_RTPThreshold  float64
	RtpAdjust_KillRateValue float64
}
type HitData struct {
	AttackRTP  float64
	TargetOdds float64
	MaxHit     int
	ChargeDrop bool
	MapBet     int32
}

func (modle *MathModel) GetPlayerCurRTP(player *Player) float64 {
	return float64(player.DBPlayer.TotalWin / player.DBPlayer.TotalExpenditure)
}

// 取得普攻擊殺率
func (model *MathModel) GetAttackKP(hitData HitData, player *Player) (float64, int64) {

	attackRTP := hitData.AttackRTP
	if hitData.ChargeDrop { // 需要把普通攻擊的部分RTP分給技能充能掉落時
		attackRTP -= model.SpellSharedRTP
	}
	if attackRTP <= 0 {
		log.Errorf("%s GetAttackKP錯誤 attackRTP<=0", logger.LOG_GameMath)
		return 0, 0
	}
	hitData.AttackRTP = attackRTP
	return model.getKPandAddPTBuffer(hitData, player)
}

// 取得技能擊殺率
func (model *MathModel) GetSpellKP(hitData HitData, player *Player) (float64, int64) {
	return model.getKPandAddPTBuffer(hitData, player)
}

func (model *MathModel) getKPandAddPTBuffer(hitData HitData, player *Player) (float64, int64) {

	// ====================點數暫存(Point Buffer)

	rewardPoint := hitData.TargetOdds * float64(hitData.MapBet)                    // 計算擊殺此怪會獲得的點數
	originalKP := hitData.AttackRTP / hitData.TargetOdds / float64(hitData.MaxHit) // 計算原始擊殺率
	pointBuffer := player.DBPlayer.PointBuffer
	// log.Infof("PointBuffer修正前=======pointBufer: %v KP: %v ", pointBuffer, originalKP)
	gainKP := float64(0) // 計算點數溢位獲得的擊殺率
	if rewardPoint != 0 {
		gainKP = float64(pointBuffer) / rewardPoint // 計算點數溢位獲得的擊殺率
	}
	if pointBuffer > 0 { // 點數溢位大於0代表要增加玩家擊殺率
	} else if pointBuffer < 0 { // 點數溢位小於0代表要降低玩家擊殺率
		gainKP = -gainKP
	}
	kp := originalKP + gainKP

	if kp > 1 { // 擊殺率大於1時處理
		overflowKP := kp - 1
		pointBuffer = int64(overflowKP * rewardPoint)
		kp = 1
	} else if kp < 0 { // 擊殺率小於0時處理
		overflowKP := -kp
		pointBuffer = int64(overflowKP * rewardPoint)
		kp = 0
	} else { // 擊殺率在0~1之間處理
		pointBuffer = 0
	}
	ptBufferChanged := pointBuffer - player.DBPlayer.PointBuffer // PointBuffer改變值
	// log.Infof("PointBuffer修正後=======pointBufer: %v KP: %v pt改變值: %v", pointBuffer, kp, ptBufferChanged)

	// ====================RTP校正(RTP Adjust)
	adjustRTP := true
	if model.RtpAdjust_KillRateValue < 0 || model.RtpAdjust_KillRateValue > 1 {
		adjustRTP = false
		log.Errorf("%s RtpAdjust_KillRateValue填錯: %v 此值應該介於0~1", logger.LOG_GameMath, model.RtpAdjust_KillRateValue)
	}
	if model.RtpAdjust_RTPThreshold < 0 {
		adjustRTP = false
	}
	if adjustRTP {
		// log.Infof("RTP校正前=======KP: %v 總贏: %v 總花費: %v", kp, player.DBPlayer.TotalWin, player.DBPlayer.TotalExpenditure)
		playerRTP := float64(player.DBPlayer.TotalWin) / float64(player.DBPlayer.TotalExpenditure) // 計算玩家實際RTP
		// log.Infof("RTP差: %v", model.GameRTP-playerRTP)
		if math.Abs(model.GameRTP-playerRTP) >= model.RtpAdjust_RTPThreshold { // RTP差>=RTP校正閥值才考慮RTP校正
			expectedTotalWin := float64(player.DBPlayer.TotalExpenditure) * model.GameRTP
			pointDiff := expectedTotalWin - float64(player.DBPlayer.TotalWin) // 計算玩家分差(玩家總贏與期望總贏差)
			pointDiffThreshold := rewardPoint * model.RtpAdjust_KillRateValue
			// log.Infof("分差: %v 分差校正閥值: %v 期望總贏: %v", pointDiff, pointDiffThreshold, expectedTotalWin)
			if math.Abs(pointDiff) >= pointDiffThreshold { // 分差 >= 分差校正閥值才進行RTP校正
				// 進行RTP校正
				if playerRTP < model.GameRTP {
					kp += model.RtpAdjust_KillRateValue
				} else {
					kp -= model.RtpAdjust_KillRateValue
				}
			}
		}
		// log.Infof("RTP校正後=======KP: %v", kp)
	} else {
		// log.Infof("不校正RTP=======KP: %v", kp)
	}

	return kp, ptBufferChanged
}

// 取得普攻擊殺掉落英雄技能機率
func (model *MathModel) GetHeroSpellDropP_AttackKilling(spellRTP float64, targetOdds float64) float64 {
	if spellRTP <= model.SpellSharedRTP {
		log.Errorf("%s HeroSpellDropP錯誤 spellRTP<=model.AttackRTP", logger.LOG_GameMath)
		return 0
	}
	attackRTP := model.GameRTP - model.SpellSharedRTP

	p := ((model.GameRTP - attackRTP) / (spellRTP - attackRTP)) / (model.GameRTP / targetOdds)
	// log.Errorf("DropP: %v  GameRTP: %v  attackRTP: %v  spellRTP: %v  targetOdds: %v", p, model.GameRTP, attackRTP, spellRTP, targetOdds)
	// 掉落率大於1時處理
	if p > 1 {
		p = 1
		log.Infof("%s GetHeroSpellDropP_AttackKilling的p>1", logger.LOG_GameMath)
	}
	return p
}
