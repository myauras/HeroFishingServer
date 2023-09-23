package main

import (
	logger "matchgame/logger"

	log "github.com/sirupsen/logrus"

	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	FirebaseFunction "majampachinkogame/Firebase"
	mainGame "majampachinkogame/Game"
	"majampachinkogame/Packet"
	"net"
	"os"
	"strings"
	"time"

	serverSDK "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

const (
	AuthLock = true
)

const ALLOW_PLAYER_NUMBER int = 4

var RoomName string
var connectionTokens []string
var EnvVersion string

func main() {
	go signalListen()

	port := flag.String("port", "7654", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	EnvVersion = *flag.String("Version", "Dev", "version setting")
	if ep := os.Getenv("Version"); ep != "" {
		EnvVersion = ep
	}

	s, err := sdk.NewSDK()
	if err != nil {
		log.Errorf("%s Could not connect to sdk: %v.\n", logger.LOG_Main, err)
	}

	waitMetaData := make(chan *mainGame.GameRoom)

	roomInit := false
	var matchmakerPodName string
	var dbMapID string
	var gsLoadDone *serverSDK.GameServer
	var playerIDs [ALLOW_PLAYER_NUMBER]string
	s.WatchGameServer(func(gs *serverSDK.GameServer) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%s Could not connect to sdk: %v.\n", logger.LOG_Main, err)
				exit(s)
			}
		}()

		if !roomInit && gs.ObjectMeta.Labels["RoomName"] != "" {
			log.Infof("%s Start room init!", logger.LOG_Main)
			matchmakerPodName = gs.ObjectMeta.Labels["MatchmakerPodName"]
			var pIDs [ALLOW_PLAYER_NUMBER]string
			for i := 0; i < ALLOW_PLAYER_NUMBER; i++ {
				pIDs[i] = gs.ObjectMeta.Labels[fmt.Sprintf("player%d", i)]
				playerIDs[i] = pIDs[i]
			}

			dbMapID = gs.ObjectMeta.Labels["MapID"]
			// roomGameDataSnap, ok := FirebaseFunction.GetRoomGameData(dbMapID)
			// if !ok {
			// 	return
			// }
			// var gameSetting mainGame.GameSetting
			// err = roomGameDataSnap.DataTo(&gameSetting)
			// if err != nil {
			// 	return
			// }
			roomInit = true
			gsLoadDone = gs
			RoomName = gs.ObjectMeta.Labels["RoomName"]
			log.Infof("%s ==============InitGameRoom==============", logger.LOG_Main)
			log.Infof("%s MatchmakerPodName: %s", logger.LOG_Main, matchmakerPodName)
			log.Infof("%s RoomName: %s", logger.LOG_Main, RoomName)
			log.Infof("%s PlayerIDs: %s", logger.LOG_Main, pIDs)
			// fmt.Println("Init gameSetting: ", gameSetting)
			// mainGame.CheckOutCheatData(&gameSetting)
			mainGame.InitGameRoom(dbMapID, RoomName, pIDs, playerIDs, nil, waitMetaData, gs.ObjectMeta.Name)
			log.Infof("%s Init Game Room Success", logger.LOG_Main)
		} else {
			if matchmakerPodName != "" && gs.ObjectMeta.Labels["MatchmakerPodName"] != "" && matchmakerPodName != gs.ObjectMeta.Labels["MatchmakerPodName"] {
				// 要改成atlas function版本
				// FirebaseFunction.WriteErrorLog(map[string]interface{}{
				// 	"ErrorID":    "ALLOCATE ERROR",
				// 	"Message":    "Agones has allocate error in parelle.",
				// 	"CreateTime": time.Now(),
				// })
			}
		}
	})

	fmt.Println("majamPachinkoGame start waitMetaData over")

	//log.Print("Starting Health Ping")
	stop := make(chan struct{})
	go doHealth(s, stop)

	//log.Print("Marking this server as ready")
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	//等收到allocate拿到房間資訊後才開啟socket連線
	gameRoom := <-waitMetaData
	fmt.Println("gameRoom := <-waitMetaData OK")
	close(waitMetaData)
	fmt.Println("OpenTCP")
	go OpenConnectTCP(s, stop, ":"+*port, gameRoom)
	fmt.Println("OpenUDP")
	go OpenConnectUDP(s, stop, ":"+*port, gameRoom)
	fmt.Println("LogPlayingGame")
	FirebaseFunction.CreateGameRoomByRoomName(gsLoadDone.Status.Address, gsLoadDone.Status.Ports[0].Port, gsLoadDone.ObjectMeta.Labels["roomName"], playerIDs, dbMapID, gsLoadDone.ObjectMeta.Name)
	endGame := make(chan struct{})
	fmt.Println("StartRun")
	gameRoom.StartRun(stop, endGame)
	fmt.Println("StartRun OVER")
	select {
	case <-stop:
		//TODO Should make a error log.
		FirebaseFunction.DeleteGameRoom(RoomName)
		fmt.Println("<-stop trigger")
		exit(s)
		return
	case <-endGame:
		//TODO write result to firebase
		FirebaseFunction.DeleteGameRoom(RoomName)
		fmt.Println("<-endGame trigger")
		DelayClose(60*time.Second, s, stop)
	}
	<-stop
	exit(s)
}

// 偵測SIGTERM/SIGKILL的終止訊號，偵測到就刪除遊戲房資料並寫log
func signalListen() {
	ctx := signals.NewSigKillContext()
	<-ctx.Done()
	// FirebaseFunction.DeleteGameRoom(documentID)
	log.Infof("%s Exit signal received. Shutting down.", logger.LOG_Main)
	os.Exit(0)
}

func OpenConnectTCP(s *sdk.SDK, stop chan struct{}, address string, gameRoom *mainGame.GameRoom) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("OpenConnectTCP error: %v.", err)
			stop <- struct{}{}
		}
	}()
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Panicf("Could not start tcp server: %v", err)
	}
	defer ln.Close() // nolint: errcheck

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Unable to accept incoming tcp connection: %v", err)
			continue
		}
		go handleConnectionTCP(conn, stop, gameRoom)
	}
}

func OpenConnectUDP(s *sdk.SDK, stop chan struct{}, address string, gameRoom *mainGame.GameRoom) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("OpenConnectUDP error: %v.", err)
			stop <- struct{}{}
		}
	}()
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		log.Panicf("Could not start udp server: %v", err)
	}
	defer conn.Close() // nolint: errcheck
	for {
		b := make([]byte, 1024)
		n, sender, err := conn.ReadFrom(b)
		if err != nil || n <= 0 {
			log.Printf("Could not read from udp stream: %v", err)
			continue
		}
		txt := strings.TrimSpace(string(b[:n]))
		//log.Printf("Received packet from %v: %v", sender.String(), txt)
		hasToken := false
		for _, t := range connectionTokens {
			//log.Printf("connectionTokens : %s", t)
			if t == txt {
				hasToken = true
			}
		}
		if hasToken {
			//log.Printf("Start Update UDP Message.")
			go handleConnectionUDP(conn, stop, sender, gameRoom)
		}
	}
}

// handleConnectionTCP services a single tcp connection to the server
func handleConnectionTCP(conn net.Conn, stop chan struct{}, gameRoom *mainGame.GameRoom) {
	remoteAddr := conn.RemoteAddr().String()
	//log.Printf("Client %s connected", conn.RemoteAddr().String())
	defer conn.Close()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("handleConnectionTCP error: %v.", err)
		}
	}()
	isAuth := false
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(1 * time.Minute))
	for {
		select {
		case <-stop:
			//被強制終止
			return
		default:
			//Nothing go thround non-blocking
		}
		packet, err := Packet.ReadPacket(decoder)
		if err != nil {
			//log.Printf("Conn : %v", conn)
			gameRoom.PlayerLeave(conn)
			return
		}
		fmt.Printf("Recv: %s from %s \n", packet.Command, remoteAddr)

		if AuthLock {
			//除了Auth指令進來未驗證都擋掉
			if EnvVersion == "Dev" {
				if !isAuth && packet.Command != "Auth" && packet.Command != "Tester" {
					log.Println("UnAuth command", packet.Command)
					return
				}
			} else {
				if !isAuth && packet.Command != "Auth" && packet.Command != "Tester" {
					log.Println("UnAuth command", packet.Command)
					return
				}
			}
		}

		if packet.Command == "Tester" {
			authContent := Packet.AuthCMD{}
			if ok := authContent.Parse(packet.Content); !ok {
				log.Printf("Pare AuthCMD Failed.")
				return
			}
			isAuth = true
			secretKey := generateSecureToken(32)
			err = Packet.SendPacket(encoder, &Packet.Packet{
				Command:  "ReAuth",
				PacketID: packet.PacketID,
				Content: &Packet.ReAuthCMD{
					IsAuth:   true,
					TokenKey: secretKey,
				},
			})
			if err != nil {
				return
			}
			defer removeConnectionToken(secretKey)
			connectionTokens = append(connectionTokens, secretKey)
			gameRoom.PlayerJoin(conn, encoder, decoder, authContent.Token)
		} else if packet.Command == "Auth" {
			authContent := Packet.AuthCMD{}
			if ok := authContent.Parse(packet.Content); !ok {
				log.Printf("Pare AuthCMD Failed.")
				return
			}
			token, err := FirebaseFunction.VerifyIDToken(authContent.Token)
			if err != nil {
				log.Printf("error verifying ID token: %v\n", err)
			} else {
				//log.Printf("Verified ID token: %v\n", token.UID)
				isAuth = true
				secretKey := generateSecureToken(32)
				err = Packet.SendPacket(encoder, &Packet.Packet{
					Command:  "ReAuth",
					PacketID: packet.PacketID,
					Content: &Packet.ReAuthCMD{
						IsAuth:   true,
						TokenKey: secretKey,
					},
				})
				if err != nil {
					return
				}
				defer removeConnectionToken(secretKey)
				connectionTokens = append(connectionTokens, secretKey)
				gameRoom.PlayerJoin(conn, encoder, decoder, token.UID)
				continue
			}
			err = Packet.SendPacket(encoder, &Packet.Packet{
				Command:  "ReAuth",
				PacketID: packet.PacketID,
				ErrMsg:   err.Error(),
				Content: &Packet.ReAuthCMD{
					IsAuth: false,
				},
			})
			if err != nil {
				return
			}
		} else {
			err = gameRoom.HandleMessage(conn, packet, stop)
			if err != nil {
				log.Printf("GameRoom Handle Message Error : %s", err.Error())
				gameRoom.PlayerLeave(conn)
				return
			}
		}
	}
}

func handleConnectionUDP(conn net.PacketConn, stop chan struct{}, addr net.Addr, gameRoom *mainGame.GameRoom) {
	timer := time.NewTicker(TIME_UPDATE_INTERVAL_MS * time.Millisecond)
	for {
		select {
		case <-stop:
			//被強制終止
			return
		case <-timer.C:
			sendData, err := json.Marshal(&Packet.Packet{
				Command:  "UDP_UPDATE",
				PacketID: -1,
				Content: mainGame.UdpUpdatePacket{
					ServerTime: gameRoom.ServerTime,
				},
			})
			if err != nil {
				log.Printf("Error Parse send UDP message. %s", err.Error())
				continue
			}
			sendData = append(sendData, '\n')
			_, sendErr := conn.WriteTo(sendData, addr)
			if sendErr != nil {
				log.Printf("Error send UDP message. %s", sendErr.Error())
				continue
			}
			//log.Printf("Send Data to %v", addr.String())
		}
	}
}

// exit shutdowns the server
func exit(s *sdk.SDK) {
	log.Printf("Received EXIT command. Exiting.")
	// This tells Agones to shutdown this Game Server
	if err := s.Shutdown(); err != nil {
		log.Printf("Could not call shutdown: %v", err)
	}
	os.Exit(0)
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Printf("Could not send health ping: %v", err)
		}
		select {
		case <-stop:
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}

func DelayClose(delay time.Duration, sdk *sdk.SDK, stop chan struct{}) {
	timer1 := time.NewTimer(delay)
	<-timer1.C
	// This tells Agones to shutdown this Game Server
	if err := sdk.Shutdown(); err != nil {
		log.Printf("Could not call shutdown: %v", err)
	}
	stop <- struct{}{}
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func removeConnectionToken(token string) {
	index := -1
	for i, v := range connectionTokens {
		if v == token {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	after := append(connectionTokens[:index], connectionTokens[index+1:]...)
	connectionTokens = after
}
