package main

import (
	"crypto/rand"
	logger "matchgame/logger"
	gSetting "matchgame/setting"
	"sync"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"

	"encoding/hex"
	"encoding/json"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"matchgame/game"
	"matchgame/packet"
	"net"
	"time"
	// sdk "agones.dev/agones/sdks/go"
)

// 開啟TCP連線
func openConnectTCP(stop chan struct{}, src string) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%s OpenConnectTCP error: %v.\n", logger.LOG_Main, err)
			stop <- struct{}{}
		}
	}()
	tcpListener, err := net.Listen("tcp", src)
	if err != nil {
		log.Errorf("%s (TCP)偵聽失敗: %v.\n", logger.LOG_Main, err)
	}
	defer tcpListener.Close()
	log.Infof("%s (TCP)開始偵聽 %s", logger.LOG_Main, src)
	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Errorf("%s Unable to accept incoming tcp connection: %v.\n", logger.LOG_Main, err)
			continue
		}
		go handleConnectionTCP(conn, stop)
	}
}

// 處理TCP連線封包
func handleConnectionTCP(conn net.Conn, stop chan struct{}) {
	remoteAddr := conn.RemoteAddr().String()

	// log.Infof("%s Client %s connected", logger.LOG_Main, conn.RemoteAddr().String())
	defer conn.Close()
	defer func() {
		log.Infof("%s 關閉handleConnectionTCP", logger.LOG_Main)
		// if err := recover(); err != nil {
		// 	// log.Errorf("%s (TCP)handleConnectionTCP錯誤: %v.", logger.LOG_Main, err)
		// }
	}()
	isAuth := false
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(gSetting.TCP_CONN_TIMEOUT_SEC * time.Second))

	packReadChan := &gSetting.LoopChan{
		StopChan:      make(chan struct{}, 1),
		ChanCloseOnce: sync.Once{},
	}

	// 封包處理
	for {
		select {
		case <-stop:
			log.Errorf("%s (TCP)強制終止連線", logger.LOG_Main)
			packReadChan.ClosePackReadStopChan()
			return
		case <-packReadChan.StopChan:
			log.Infof("%s (TCP)關閉封包讀取", logger.LOG_Main)
			return // 終止goroutine
		default:
			pack, err := packet.ReadPack(decoder)
			if err != nil {
				log.Infof("%s (TCP)packReadReadResult錯誤: %v.", logger.LOG_Main, err)
				packReadChan.ClosePackReadStopChan()
				return
			}
			log.Infof("%s (TCP)收到來自%s 的命令: %s \n", logger.LOG_Main, remoteAddr, pack.CMD)

			//未驗證前，除了Auth指令進來其他都擋掉
			if !isAuth && pack.CMD != packet.AUTH {
				log.Infof("%s 收到未驗證的封包", logger.LOG_Main)
				packReadChan.ClosePackReadStopChan()
				return
			}
			if pack.CMD == packet.AUTH {
				authContent := packet.Auth{}
				if ok := authContent.Parse(pack.Content); !ok {
					log.Errorf("%s 反序列化AUTH封包失敗", logger.LOG_Main)
					continue
				}
				// 像mongodb atlas驗證token並取得playerID 有通過驗證後才處理後續
				playerID, authErr := mongo.PlayerVerify(authContent.Token)
				// 驗證失敗
				if authErr != nil || playerID == "" {
					log.Errorf("%s 玩家驗證錯誤: %v", logger.LOG_Main, authErr)
					_ = packet.SendPack(encoder, &packet.Pack{
						CMD:    packet.AUTH_TOCLIENT,
						PackID: pack.PackID,
						ErrMsg: "玩家驗證錯誤",
						Content: &packet.Auth_ToClient{
							IsAuth: false,
						},
					})
				}
				isAuth = true
				var player game.Player
				// 斷線重連檢測
				reConnection := false
				for _, v := range game.MyRoom.Players {
					if v == nil {
						continue
					}
					if v.DBPlayer.ID == playerID {
						log.Infof("玩家(%v)斷線重連", playerID)
						reConnection = true
						player = *v
						break
					}
				}

				// 建立udp socket連線Token
				newConnToken := generateSecureToken(32)

				if !reConnection { // 不是斷線重連就建立玩家資料

					var dbPlayer mongo.DBPlayer
					getPlayerDocErr := mongo.GetDocByID(mongo.ColName.Player, playerID, &dbPlayer)
					if getPlayerDocErr != nil {
						log.Errorf("%s DBPlayer資料錯誤: %v", logger.LOG_Main, getPlayerDocErr)
						_ = packet.SendPack(encoder, &packet.Pack{
							CMD:    packet.AUTH_TOCLIENT,
							PackID: pack.PackID,
							ErrMsg: "DBPlayer資料錯誤",
							Content: &packet.Auth_ToClient{
								IsAuth: false,
							},
						})
					}

					// 建立RedisDB Player
					redisPlayer, redisPlayerErr := redis.CreatePlayerData(dbPlayer.ID, dbPlayer.Point, dbPlayer.PointBuffer, dbPlayer.TotalWin, dbPlayer.TotalExpenditure, dbPlayer.HeroExp, dbPlayer.SpellCharges, dbPlayer.Drops)
					if redisPlayerErr != nil {
						log.Errorf("%s 建立RedisPlayer錯誤: %v", logger.LOG_Main, redisPlayerErr)
						_ = packet.SendPack(encoder, &packet.Pack{
							CMD:    packet.AUTH_TOCLIENT,
							PackID: pack.PackID,
							ErrMsg: "建立RedisPlayer錯誤",
							Content: &packet.Auth_ToClient{
								IsAuth: false,
							},
						})
					}

					// 將該玩家monogoDB上的redisSync設為false
					updatePlayerBson := bson.D{
						{Key: "redisSync", Value: false},
					}
					_, updateErr := mongo.UpdateDocByBsonD(mongo.ColName.Player, dbPlayer.ID, updatePlayerBson)
					if updateErr != nil {
						log.Errorf("%s 更新玩家 %s DB資料錯誤: %v", logger.LOG_Main, dbPlayer.ID, updateErr)
					}

					// 將玩家加入遊戲房
					player = game.Player{
						DBPlayer:     &dbPlayer,
						RedisPlayer:  redisPlayer,
						LastUpdateAt: time.Now(),
						PlayerBuffs:  []packet.PlayerBuff{},
						ConnTCP: &gSetting.ConnectionTCP{
							Conn:       conn,
							MyLoopChan: packReadChan,
							Encoder:    encoder,
							Decoder:    decoder,
						},
						ConnUDP: &gSetting.ConnectionUDP{
							ConnToken: newConnToken,
						},
					}
					joined := game.MyRoom.JoinPlayer(&player)
					if !joined {
						log.Errorf("%s 玩家加入房間失敗", logger.LOG_Main)
						packReadChan.ClosePackReadStopChan()
						return
					}

				} else { // 斷線重連時使用已存在的玩家資料, 不須重建資料
					player.ConnTCP.Conn = conn
					player.ConnUDP.ConnToken = newConnToken
				}
				// 回送client
				err = packet.SendPack(encoder, &packet.Pack{
					CMD:    packet.AUTH_TOCLIENT,
					PackID: pack.PackID,
					Content: &packet.Auth_ToClient{
						IsAuth:    true,
						ConnToken: player.ConnUDP.ConnToken,
						Index:     player.Index,
					},
				})
				if err != nil {
					continue
				}

			} else {
				err = game.MyRoom.HandleTCPMsg(conn, pack)
				if err != nil {
					log.Errorf("%s (TCP)處理GameRoom封包錯誤: %v\n", logger.LOG_Main, err.Error())
					game.MyRoom.KickPlayer(conn, "處理GameRoom封包錯誤")
					continue
				}
			}
		}

	}
}

// 產生連線驗證Token
func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
