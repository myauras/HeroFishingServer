package main

import (
	"encoding/json"
	"flag"
	"herofishingGoModule/gameJson"
	"herofishingGoModule/k8s"
	mongo "herofishingGoModule/mongo"
	"herofishingGoModule/redis"
	"herofishingGoModule/setting"
	logger "matchmaker/logger"
	"matchmaker/packet"
	matchmakerSetting "matchmaker/setting"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

var Env string                    // 環境版本
var SelfPodName string            // K8s上所屬的Pod名稱
var Receptionist RoomReceptionist // 房間接待員

func main() {
	// 設定日誌級別
	log.SetLevel(log.InfoLevel)
	// 設定日誌輸出，預設為標準輸出
	log.SetOutput(os.Stdout)
	// 自定義時間格式，包含毫秒
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Main Crash: %v", r)
		}
	}()

	log.Infof("%s ==============MATCHMAKER 啟動==============", logger.LOG_Main)
	// // 顯示Image版本
	// if imgVer := os.Getenv("ImgVer"); imgVer != "" {
	// 	log.Infof("%s Image版本為: %s", logger.LOG_Main, imgVer)
	// } else {
	// 	log.Errorf("%s 取不到環境變數: ImgVer", logger.LOG_Main)
	// }
	// 設定Port
	port := flag.String("port", "32680", "The port to listen to tcp traffic on")
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = &envPort
	}

	// 設定環境版本
	Env = *flag.String("Env", "Dev", "Env setting")
	if envEnv := os.Getenv("Env"); envEnv != "" {
		Env = envEnv
	}
	log.Infof("%s Env: %s", logger.LOG_Main, Env)

	// 設定K8s上所屬的Pod名稱
	SelfPodName = *flag.String("MY_POD_NAME", "myPodName", "Pod Name")
	if envSelfPodName := os.Getenv("MY_POD_NAME"); envSelfPodName != "" {
		SelfPodName = envSelfPodName
	}
	// 取得API public Key
	mongoAPIPublicKey := os.Getenv("MongoAPIPublicKey")
	log.Infof("%s MongoAPIPublicKey: %s", logger.LOG_Main, mongoAPIPublicKey)

	// 取得API private Key
	mongoAPIPrivateKey := os.Getenv("MongoAPIPrivateKey")
	log.Infof("%s MongoAPIPrivateKey: %s", logger.LOG_Main, mongoAPIPrivateKey)

	// 取得MongoDB帳密
	mongoUser := os.Getenv("MongoUser")
	mongoPW := os.Getenv("MongoPW")
	log.Infof("%s MongoUser: %s", logger.LOG_Main, mongoUser)
	log.Infof("%s mongoPW: %s", logger.LOG_Main, mongoPW)

	// 初始化redisDB
	redis.Init()

	InitGameJson() // 初始化遊戲Json資料

	// 初始化MongoDB設定
	initMonogo(mongoAPIPublicKey, mongoAPIPrivateKey, mongoUser, mongoPW)

	// 初始化Agones
	InitAgones()

	// 取Loadbalancer分配給此pod的對外IP並寫入資料庫
	log.Infof("%s 取Loadbalancer分配給此pod的對外IP.\n", logger.LOG_Main)
	for {
		// 因為pod啟動後Loadbalancer並不會立刻就分配好ip(會有延遲) 所以每5秒取一次 直到取到ip才往下跑
		time.Sleep(5 * time.Second) // 每5秒取一次ip
		ip, getIPErr := getExternalIP()
		if getIPErr != nil {
			// 取得ip失敗
			break
		}
		if ip != "" {
			log.Infof("%s 取得對外IP成功: %s .\n", logger.LOG_Main, ip)
			log.Infof("%s 開始寫入對外ID到DB.\n", logger.LOG_Main)
			setExternalIP(ip) // 寫入對外ID到DB中
			break
		}
	}

	// 偵聽TCP封包
	src := ":" + *port
	tcpListener, err := net.Listen("tcp", src)
	if err != nil {
		log.Errorf("%s Listen error %s.\n", logger.LOG_Main, err.Error())
	}
	defer tcpListener.Close()
	log.Infof("%s TCP server start and listening on %s", logger.LOG_Main, src)

	// 初始化配房者
	log.Infof("%s 初始化配房者.\n", logger.LOG_Main)
	Receptionist.Init()
	log.Infof("%s 初始化配房者完成.\n", logger.LOG_Main)

	// tcp連線
	log.Infof("%s ==============MATCHMAKER啟動完成============== .\n", logger.LOG_Main)
	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Errorf("%s Connection error %s.\n", logger.LOG_Main, err)
		}
		go handleConnectionTCP(conn)
	}
}

// 初始化遊戲Json資料
func InitGameJson() {
	log.Infof("%s 開始初始化GameJson", logger.LOG_Main)
	err := gameJson.Init(Env)
	if err != nil {
		log.Errorf("%s 初始化GameJson失敗: %v", logger.LOG_Main, err)
		return
	}
}

// 初始化MongoDB設定
func initMonogo(mongoAPIPublicKey string, mongoAPIPrivateKey string, user string, pw string) {
	log.Infof("%s 初始化mongo開始", logger.LOG_Main)
	mongo.Init(mongo.InitData{
		Env:           Env,
		APIPublicKey:  mongoAPIPublicKey,
		APIPrivateKey: mongoAPIPrivateKey,
	}, user, pw)
	log.Infof("%s 初始化mongo完成", logger.LOG_Main)
}

// 寫入對外IP到DB中
func setExternalIP(ip string) {
	// 設定要更新的資料
	data := bson.D{
		{Key: "matchmakerIP", Value: ip},
	}
	// 更新資料
	_, err := mongo.UpdateDocByBsonD(mongo.ColName.GameSetting, "GameState", data)
	if err != nil {
		log.Errorf("%s SetExternalID失敗: %v", logger.LOG_Main, err)
		return
	}
}

// 取Loadbalancer分配給此pod的對外IP
func getExternalIP() (string, error) {
	ip, err := k8s.GetLoadBalancerExternalIP(setting.NAMESPACE_MATCHERSERVER, setting.MATCHMAKER)
	if err != nil {
		log.Errorf("%s GetLoadBalancerExternalIP error: %v.\n", logger.LOG_Main, err)
	}
	return ip, err
}

// 處理TCP封包
func handleConnectionTCP(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Infof("%s Client connected from: %s", logger.LOG_Main, remoteAddr)
	defer conn.Close()

	player := roomPlayer{
		id:     "",
		isAuth: false,
		connTCP: ConnectionTCP{
			Conn:    conn,
			Encoder: json.NewEncoder(conn),
			Decoder: json.NewDecoder(conn),
		},
		dbMapID: "",
		room:    nil,
	}

	go disconnectCheck(&player)

	for {
		pack, err := packet.ReadPack(player.connTCP.Decoder)
		if err != nil {
			return
		}

		log.Infof("%s Receive %s from %s", logger.LOG_Main, pack.CMD, remoteAddr)

		//收到Auth以外的命令如果未驗證就都擋掉
		if !player.isAuth && pack.CMD != packet.AUTH {

			log.WithFields(log.Fields{
				"cmd":     pack.CMD,
				"address": remoteAddr,
			}).Infof("%s UnAuthed CMD", logger.LOG_Main)
			return
		}

		// 封包處理
		switch pack.CMD {
		case packet.AUTH:
			packHandle_Auth(pack, &player)
		case packet.CREATEROOM:
			log.Infof("%s =========CREATEROOM=========", logger.LOG_Main)
			packHandle_CreateRoom(pack, &player, remoteAddr)
		default:
			log.Errorf("%s got unknow Pack CMD: %s", logger.LOG_Main, pack.CMD)
			return
		}

	}
}

// 處理封包-帳戶驗證
func packHandle_Auth(pack packet.Pack, player *roomPlayer) {
	authContent := packet.Auth{}
	if ok := authContent.Parse(pack.Content); !ok {
		log.Errorf("%s Parse AuthCMD failed", logger.LOG_Main)
		return
	}

	// 像mongodb atlas驗證token並取得playerID
	playerID, authErr := mongo.PlayerVerify(authContent.Token)
	// 驗證失敗
	if authErr != nil || playerID == "" {
		log.Errorf("%s Player verify failed: %v", logger.LOG_Main, authErr)
		_ = packet.SendPack(player.connTCP.Encoder, &packet.Pack{
			CMD:    packet.AUTH_TOCLIENT,
			PackID: pack.PackID,
			ErrMsg: "Auth toekn驗證失敗",
			Content: &packet.AuthC_ToClient{
				IsAuth: false,
			},
		})
	}

	// 驗證通過
	log.Infof("%s Player verify success, playerID: %s", logger.LOG_Main, playerID)
	player.isAuth = true
	err := packet.SendPack(player.connTCP.Encoder, &packet.Pack{
		CMD:    packet.AUTH_TOCLIENT,
		PackID: pack.PackID,
		Content: &packet.AuthC_ToClient{
			IsAuth: true,
		},
	})
	if err != nil {
		return
	}
}

// 處理封包-開遊戲房
func packHandle_CreateRoom(pack packet.Pack, player *roomPlayer, remoteAddr string) {
	createRoomCMD := packet.CreateRoom{}
	if ok := createRoomCMD.Parse(pack.Content); !ok {
		log.Infof("%s Parse CreateRoom failed", logger.LOG_Main)
		return
	}

	var dbMap mongo.DBMap
	err := mongo.GetDocByID(mongo.ColName.Map, createRoomCMD.DBMapID, &dbMap)
	if err != nil {
		log.Errorf("%s Failed to get dbmap doc: %v", logger.LOG_Main, err)
		return
	}

	log.Infof("%s dbMap: %+v", logger.LOG_Main, dbMap)

	player.id = createRoomCMD.CreaterID
	player.dbMapID = dbMap.ID

	// 根據DB地圖設定來開遊戲房
	switch dbMap.MatchType {
	case setting.MatchType.Quick: // 快速配對
		var isNewRoom bool
		player.room, isNewRoom = Receptionist.JoinRoom(pack.PackID, dbMap, player)
		if player.room == nil {
			log.WithFields(log.Fields{
				"dbMap":  dbMap,
				"player": player,
			}).Errorf("%s Join quick match room failed", logger.LOG_Main)
			// 回送房間建立失敗封包
			sendCreateRoomCMD_Reply(*player, pack, "Join quick match room failed")
			return
		}
		gs := player.room.gameServer

		// 如果是加入已存在的房間就直接回送封包, 如果是建立新房間就等待Cluster把Matchgame建立好後PubMsg接收到時才回送封包
		if !isNewRoom {
			packErr := packet.SendPack(player.connTCP.Encoder, &packet.Pack{
				CMD:    packet.CREATEROOM_TOCLIENT,
				PackID: pack.PackID,
				Content: &packet.CreateRoom_ToClient{
					CreaterID:     player.room.creater.id,
					PlayerIDs:     player.room.GetPlayerIDs(),
					DBMapID:       player.room.dbMapID,
					DBMatchgameID: player.room.dbMatchgameID,
					IP:            gs.Status.Address,
					Port:          gs.Status.Ports[0].Port,
					PodName:       gs.ObjectMeta.Name,
				},
			})
			if packErr != nil {
				return
			}
		}

	default:

		log.WithFields(log.Fields{
			"dbMap.matchType": dbMap.MatchType,
			"remoteAddr":      remoteAddr,
		}).Errorf("%s Undefined match type", logger.LOG_Main)

		// 回送房間建立失敗封包
		if err := sendCreateRoomCMD_Reply(*player, pack, "Undefined match type"); err != nil {
			return
		}
	}
}

// 斷線玩家偵測
func disconnectCheck(p *roomPlayer) {
	time.Sleep(matchmakerSetting.DISCONNECT_CHECK_INTERVAL_SECS * time.Second) // 等待後開始跑斷線檢測迴圈
	timer := time.NewTicker(matchmakerSetting.DISCONNECT_CHECK_INTERVAL_SECS * time.Second)
	for {
		<-timer.C
		if p.room == nil || p.id == "" {
			log.Infof("%s Disconnect IP: %s , because it's life is over", logger.LOG_Main, p.connTCP.Conn.RemoteAddr().String())
			p.connTCP.Conn.Close()
			return
		}
	}
}

// 送創建房間結果封包
func sendCreateRoomCMD_Reply(player roomPlayer, p packet.Pack, log string) error {
	err := packet.SendPack(player.connTCP.Encoder, &packet.Pack{
		CMD:     packet.CREATEROOM_TOCLIENT,
		PackID:  p.PackID,
		Content: &packet.CreateRoom_ToClient{},
		ErrMsg:  log,
	})
	return err
}
