package main

import (
	// "herofishingGoModule/setting"
	logger "matchgame/logger"
	gSetting "matchgame/setting"
	"sync"

	"encoding/json"

	log "github.com/sirupsen/logrus"

	"matchgame/game"
	"matchgame/packet"
	"net"
	"time"
	// sdk "agones.dev/agones/sdks/go"
)

// 開啟UDP連線
func openConnectUDP(stop chan struct{}, src string) {
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
		if pack.CMD != packet.UPDATEGAME {
			log.Infof("%s (UDP)收到來自 %s 的命令: %s \n", logger.LOG_Main, addr.String(), pack.CMD)
		}

		// 執行命令
		if pack.CMD == packet.UDPAUTH {
			if player.ConnUDP.Conn != nil {
				log.Errorf("%s (UDP)玩家(%s)斷線重連UDP", logger.LOG_Main, player.DBPlayer.ID)
				if player.ConnUDP.Addr.String() != addr.String() { // 玩家通過ConnToken驗證但Addr有變更可能是因為Wifi環境改變
					log.Infof("%s (UDP)玩家 %s 的位置從 %s 變更為 %s \n", logger.LOG_Main, player.DBPlayer.ID, player.ConnUDP.Addr.String(), addr.String())
				}
			} else {
				go updateGameLoop(player, stop)
			}
			// 更新連線資料
			player.ConnUDP.Conn = conn
			player.ConnUDP.Addr = addr
		} else {
			if player.ConnUDP.Conn == nil || player.ConnUDP.Addr == nil {
				log.Warnf("%s (UDP)收到來自 %s(%s) 但尚未進行UDP Auth的命令: %s", logger.LOG_Main, player.DBPlayer.ID, addr, pack.CMD)
			}
			// 更新連線資料
			player.ConnUDP.Conn = conn

			switch pack.CMD {

			// ==========更新遊戲狀態==========
			case packet.UPDATEGAME:
				// log.Infof("%s 更新玩家 %s 心跳", logger.LOG_Main, player.DBPlayer.ID)
				player.LastUpdateAt = time.Now() // 更新心跳
			}
		}
	}
}

// 定時更新遊戲狀態給Client
func updateGameLoop(player *game.Player, stop chan struct{}) {
	log.Infof("%s (UDP)開始updateGameLoop", logger.LOG_Main)
	gameUpdateTimer := time.NewTicker(gSetting.GAMEUPDATE_MS * time.Millisecond)
	playerUpdateTimer := time.NewTicker(gSetting.PLAYERUPDATE_MS * time.Millisecond)
	sceneUpdateTimer := time.NewTicker(gSetting.SCENEUPDATE_MS * time.Millisecond)

	defer gameUpdateTimer.Stop()
	defer playerUpdateTimer.Stop()
	defer sceneUpdateTimer.Stop()

	loopChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}
	player.ConnUDP.MyLoopChan = loopChan

	for {
		select {
		case <-stop:
			log.Infof("強制終止玩家updateGameLoop")
			return
		case <-loopChan.StopChan:
			log.Infof("終止玩家updateGameLoop")
			return
		// ==========更新遊戲狀態==========
		case <-gameUpdateTimer.C:
			// log.Infof("game.MyRoom.GameTime: %v", game.MyRoom.GameTime)
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATEGAME_TOCLIENT,
				PackID: -1,
				Content: &packet.UpdateGame_ToClient{
					GameTime: game.MyRoom.GameTime,
				},
			})
			if err != nil {
				log.Errorf("%s (UDP)序列化UPDATEGAME封包錯誤. %s", logger.LOG_Main, err.Error())
				continue
			}
			sendData = append(sendData, '\n')
			game.MyRoom.SendPacketToPlayer_UDP(player.Index, sendData)
		// ==========更新玩家狀態==========
		case <-playerUpdateTimer.C:
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATEPLAYER_TOCLIENT,
				PackID: -1,
				Content: &packet.UpdatePlayer_ToClient{
					Players: game.MyRoom.GetPacketPlayers(),
				},
			})
			if err != nil {
				log.Errorf("%s UpdatePlayers_UDP錯誤. %s", logger.LOG_Room, err.Error())
				return
			}
			sendData = append(sendData, '\n')
			game.MyRoom.SendPacketToPlayer_UDP(player.Index, sendData)
		// ==========更新場景狀態==========
		case <-sceneUpdateTimer.C:
			sendData, err := json.Marshal(&packet.Pack{
				CMD:    packet.UPDATESCENE_TOCLIENT,
				PackID: -1,
				Content: &packet.UpdateScene_ToClient{
					Spawns:       game.MyRoom.MSpawner.Spawns,
					SceneEffects: game.MyRoom.SceneEffects,
				},
			})
			if err != nil {
				log.Errorf("%s UpdatePlayers_UDP錯誤. %s", logger.LOG_Room, err.Error())
				return
			}
			sendData = append(sendData, '\n')
			game.MyRoom.SendPacketToPlayer_UDP(player.Index, sendData)

		}

	}
}
