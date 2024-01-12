package packet

import (
	"encoding/json"

	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 封包命令列表
const (
	AUTH                  = "AUTH"                  // (TCP)身分驗證
	AUTH_TOCLIENT         = "AUTH_TOCLIENT"         // (TCP)身分驗證-送Client
	SETHERO               = "SETHERO"               // (TCP)設定玩家英雄
	SETHERO_TOCLIENT      = "SETHERO_TOCLIENT"      // (TCP)設定玩家英雄-送Client
	LEAVE                 = "LEAVE"                 // (TCP)離開遊戲房
	LEAVE_TOCLIENT        = "LEAVE_TOCLIENT"        // (TCP)離開遊戲房-送Client
	ATTACK                = "ATTACK"                // (UDP)攻擊
	ATTACK_TOCLIENT       = "ATTACK_TOCLIENT"       // (UDP)攻擊-送Client
	HIT                   = "HIT"                   // (TCP)擊中
	HIT_TOCLIENT          = "HIT_TOCLIENT"          // (TCP)擊中-送Client
	DROPSPELL             = "DROPSPELL"             // (TCP)掉落施法
	UPDATEPLAYER_TOCLIENT = "UPDATEPLAYER_TOCLIENT" // (TCP)更新玩家-送Client
	SPAWN_TOCLIENT        = "SPAWN_TOCLIENT"        // (TCP)生怪-送Client
	UDPAUTH               = "UDPAUTH"               // (UDP)身分驗證
	UPDATEGAME            = "UPDATEGAME"            // (UDP)遊戲狀態更新(太久沒收到回傳會將該玩家從房間踢出)
	UPDATEGAME_TOCLIENT   = "UPDATEGAME_TOCLIENT"   // (UDP)遊戲狀態更新-送Client(每GAMEUPDATE_MS毫秒會送一次)
	UPDATESCENE           = "UPDATESCENE"           // (TCP)場景狀態更新(玩家斷線回連時會主動送過來跟server要資料)
	UPDATESCENE_TOCLIENT  = "UPDATESCENE_TOCLIENT"  // (UDP&TCP)場景狀態更新-送Client(每SCENEUPDATE_MS毫秒會送一次 或 玩家斷線回連時主動要求時會送)
	// 測試用
	MONSTERDIE_TOCLIENT = "MONSTERDIE_TOCLIENT" // (TCP)怪物死亡時送Client
)

type Pack struct {
	CMD     string
	PackID  int
	ErrMsg  string
	Content CMDContent
}
type UDPReceivePack struct {
	CMD       string
	PackID    int
	ErrMsg    string
	ConnToken string // 收到的UPD CMD除了UDPAUTH以外都會包含ConnToken
	Content   CMDContent
}

type CMDContent interface {
}

func ReadPack(decoder *json.Decoder) (Pack, error) {
	var packet Pack
	err := decoder.Decode(&packet)

	// 寫LOG
	// log.WithFields(log.Fields{
	// 	"cmd":     packet.CMD,
	// 	"content": packet.Content,
	// 	"error":   packet.ErrMsg,
	// }).Infof("%s Read: %s", logger.LOG_Pack, packet.CMD)
	if err != nil {
		if err.Error() == "EOF" { // 玩家已經斷線
		} else {
			// 寫LOG
			log.WithFields(log.Fields{
				"error": packet.ErrMsg,
			}).Errorf("Decode packet error: %s", err.Error())
		}
	}

	return packet, err
}

func SendPack(encoder *json.Encoder, packet *Pack) error {
	err := encoder.Encode(packet)

	// // 寫LOG
	// log.WithFields(log.Fields{
	// 	"cmd":     packet.CMD,
	// 	"content": packet.Content,
	// }).Infof("%s Send packet: %s", logger.LOG_Pack, packet.CMD)

	if err != nil {
		// 寫LOG
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Errorf("%s Send packet encoder error", logger.LOG_Pack)

	}
	return err
}
