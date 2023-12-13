package main

import (
	"herofishingGoModule/setting"
	logger "matchgame/logger"
	gSetting "matchgame/setting"

	"encoding/json"

	log "github.com/sirupsen/logrus"

	"matchgame/game"
	"matchgame/packet"
	"net"
	"time"

	sdk "agones.dev/agones/sdks/go"
)

// 開啟UDP連線
func openConnectUDP(s *sdk.SDK, stop chan struct{}, src string) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectUDP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	conn, err := net.ListenPacket("udp", src)
	if err != nil {
		log.Errorf("%s (UDP)偵聽失敗: %v.\n", logger.LOG_Main, err)
	}
	defer conn.Close()
	log.Infof("%s (UDP)開始偵聽 %s", logger.LOG_Main, src)

	for {
		// 取得收到的封包
		buffer := make([]byte, 1024)
		n, addr, readBufferErr := conn.ReadFrom(buffer)
		if readBufferErr != nil {
			log.Errorf("%s (UDP)讀取封包錯誤: %v", logger.LOG_Main, readBufferErr)
			continue
		}
		if n <= 0 {
			continue
		}
		// 解析json數據
		var pack packet.UDPReceivePack
		unmarshalErr := json.Unmarshal(buffer[:n], &pack)
		if unmarshalErr != nil {
			log.Errorf("%s (UDP)解析封包錯誤: %s", logger.LOG_Main, unmarshalErr.Error())
			continue
		}
		// 玩家驗證
		player := game.MyRoom.GetPlayerByConnToken(pack.ConnToken)

		if player == nil {
			log.Errorf("%s (UDP)Token驗證失敗 來自 %s 的命令: %s \n", logger.LOG_Main, addr.String(), pack.CMD)
			continue
		}
		log.Infof("%s (UDP)收到來自 %s 的命令: %s \n", logger.LOG_Main, addr.String(), pack.CMD)

		// 執行命令
		if pack.CMD == packet.UDPAUTH {
			if player.ConnUDP.Conn != nil {
				log.Errorf("%s (UDP)此玩家已執行過UDP Auth有正在進行的updateGameLoop", logger.LOG_Main)
				continue
			}
			// 更新連線資料
			player.ConnUDP.Conn = conn
			player.ConnUDP.Addr = addr
			go updateGameLoop(player, stop)
		} else {
			if player.ConnUDP.Conn == nil || player.ConnUDP.Addr == nil {
				log.Errorf("%s (UDP)收到來自 %s(%s) 但尚未進行UDP Auth的命令: %s", logger.LOG_Main, player.DBPlayer.ID, addr, pack.CMD)
			}
			// 更新連線資料
			player.ConnUDP.Conn = conn
			if player.ConnUDP.Addr.String() != addr.String() { // 玩家通過ConnToken驗證但Addr有變更可能是因為Wifi環境改變
				log.Infof("%s (UDP)玩家 %s 的位置從 %s 變更為 %s \n", logger.LOG_Main, player.DBPlayer.ID, player.ConnUDP.Addr.String(), addr.String())
				// 更新address避免客戶端的網路位置有改變這樣對於Wifi變更的用戶體驗比較好
				// 但是要注意若之後有使用udp送重要行為 為了避免connToken被封包攔截要讓玩家需要重新通過tcp auth取新的token才是安全的作法
				player.ConnUDP.Addr = addr
			}
			switch pack.CMD {

			// ==========更新遊戲狀態==========
			case packet.UPDATEGAME:
				// log.Infof("%s 更新玩家 %s 心跳", logger.LOG_Main, player.DBPlayer.ID)
				player.LastUpdateAt = time.Now() // 更新心跳

			// ==========發動攻擊==========
			case packet.ATTACK:
				content := packet.Attack{}
				if ok := content.Parse(pack.Content); !ok {
					log.Errorf("%s parse %s failed", logger.LOG_Main, pack.CMD)
					continue
				}
				game.MyRoom.HandleAttack(player, pack, content)
			}
		}
	}
}

// 定時更新遊戲狀態給Client
func updateGameLoop(player *game.Player, stop chan struct{}) {
	log.Infof("%s (UDP)開始updateGameLoop", logger.LOG_Main)
	timer := time.NewTicker(gSetting.GAMEUPDATE_MS * time.Millisecond)
	for {
		select {
		case <-stop:
			//被強制終止
			log.Errorf("強制終止UDP")
			return
		case <-timer.C:
			if player == nil || player.ConnUDP == nil {
				return
			}
			// 定時送遊戲更新給Client
			var playerGainPoints [setting.PLAYER_NUMBER]int64
			for i, v := range game.MyRoom.Players {
				if v != nil {
					playerGainPoints[i] = v.GainPoint
				} else {
					playerGainPoints[i] = 0
				}
			}
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATEGAME_TOCLIENT,
				PackID: -1,
				Content: &packet.UpdateGame_ToClient{
					GameTime:         game.MyRoom.GameTime,
					PlayerGainPoints: playerGainPoints,
				},
			})
			if err != nil {
				log.Errorf("%s (UDP)序列化UPDATEGAME封包錯誤. %s", logger.LOG_Main, err.Error())
				continue
			}
			sendData = append(sendData, '\n')
			_, sendErr := player.ConnUDP.Conn.WriteTo(sendData, player.ConnUDP.Addr)
			if sendErr != nil {
				log.Errorf("%s (UDP)送UPDATEGAME封包錯誤 %s", logger.LOG_Main, sendErr.Error())
				continue
			}
		}
	}
}
