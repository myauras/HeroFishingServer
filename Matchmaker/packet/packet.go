package packet

import (
	"encoding/json"

	logger "matchmaker/logger"

	log "github.com/sirupsen/logrus"
)

// 封包命令列表
const (
	AUTH                = "AUTH"                // (TCP)身分驗證
	AUTH_TOCLIENT       = "AUTH_TOCLIENT"       // (TCP)身分驗證-送Client
	CREATEROOM          = "CREATEROOM"          // (TCP)建立房間
	CREATEROOM_TOCLIENT = "CREATEROOM_TOCLIENT" // (TCP)建立房間回傳
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
