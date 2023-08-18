package packet

import (
	"encoding/json"

	logger "matchmaker/Logger"

	log "github.com/sirupsen/logrus"
)

type Packet struct {
	CMD      string
	PacketID int
	ErrMsg   string
	Content  CMDContent
}

type CMDContent interface {
}

func ReadPacket(decoder *json.Decoder) (Packet, error) {
	var packet Packet
	err := decoder.Decode(&packet)
	if err == nil {
		// 寫LOG
		log.WithFields(log.Fields{
			"cmd":     packet.CMD,
			"content": packet.Content,
			"error":   packet.ErrMsg,
		}).Infof("%s Read: %s", logger.LOG_Packet, packet.CMD)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"cmd":     packet.CMD,
			"content": packet.Content,
			"error":   packet.ErrMsg,
		}).Errorf("%s Read: %s", logger.LOG_Packet, packet.CMD)
	}

	return packet, err
}

func SendPacket(encoder *json.Encoder, packet *Packet) error {
	err := encoder.Encode(packet)

	if err == nil {
		// 寫LOG
		log.WithFields(log.Fields{
			"cmd":     packet.CMD,
			"content": packet.Content,
		}).Infof("%s Send packet: %s", logger.LOG_Packet, packet.CMD)
	} else {
		// 寫LOG
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Errorf("%s Send packet encoder error", logger.LOG_Packet)
	}

	return err
}
