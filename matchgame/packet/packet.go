package packet

import (
	"encoding/json"

	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"
)

// 封包命令列表
const (
	AUTH                    = "AUTH"                    // 身分驗證(TCP)
	AUTH_REPLY              = "AUTH_REPLY"              // 身分驗證回傳(TCP)
	ACTION_SETHERO          = "ACTION_SETHERO"          // 設定玩家英雄(TCP)
	ACTION_SETHERO_REPLY    = "ACTION_SETHERO_REPLY"    // 設定玩家英雄回傳(TCP)
	UPDATE_UDP              = "UPDATE_UDP"              // 狀態更新(UDP)
	UPDATE_GAME_STATE_REPLY = "UPDATE_GAME_STATE_REPLY" // 更新遊戲狀態
	SPAWNM                  = "SPAWN"                   // 生怪(TCP)
)

type Pack struct {
	CMD     string
	PackID  int
	ErrMsg  string
	Content CMDContent
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
