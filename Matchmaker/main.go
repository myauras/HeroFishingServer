package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

var EvnVersion string             // 環境版本(Dev, Release)
var Receptionist RoomReceptionist // 房間接待員

func main() {
	log.Infof("%s ==============MATCHMAKER START==============", LOG_Main)

	port := flag.String("port", "32680", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	log.Infof("Port: %s", *port)

	EvnVersion = *flag.String("Version", "Dev", "Version setting")
	if ep := os.Getenv("Version"); ep != "" {
		EvnVersion = ep
	}
	log.Infof("EvnVersion: %s", EvnVersion)

	// 偵聽TCP封包
	src := ":" + *port
	listener, err := net.Listen("tcp", src)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer listener.Close()
	log.Infof("TCP server start and listening on %s.\n", src)

	Receptionist.Init()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Infof("Connection error %s.\n", err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	//log.Println("Client connected from: " + remoteAddr)
	defer conn.Close()

	player := roomPlayer{
		id:      "",
		isAuth:  false,
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
		mapID:   "",
		room:    nil,
	}

	defer func() {
		if player.room != nil {
			RoomReceptionist.PlayerLeaveRoom(player)
			player.LeaveRoom()
		}
		if err := recover(); err != nil {
			log.Printf("Panic Handle: %v", err)
		}
	}()

	go checkForceDisconnect(&player)

	for {
		packet, err := Packet.ReadPacket(player.Decoder)
		if err != nil {
			//fmt.Printf("Read packet error: %s\n", err.Error())
			return
		}
		fmt.Printf("Recv: %s from %s \n", packet.Command, remoteAddr)

		if AuthLock {
			//除了Auth指令進來未驗證都擋掉
			if EnvironmentVersion == "Dev" {
				if !player.IsAuth && packet.Command != "Auth" && packet.Command != "Tester" {
					log.Println("UnAuth command", packet.Command)
					return
				}
			} else {
				if !player.IsAuth && packet.Command != "Auth" && packet.Command != "Tester" {
					log.Println("UnAuth command", packet.Command)
					return
				}
			}
		}

		switch packet.Command {
		case "InnerCommander":
			err = Packet.SendPacket(player.Encoder, &Packet.Packet{
				Command: "ReCommander",
				Content: nil,
			})
			if err != nil {
				return
			}

		case "RenewAIIndex":
			AIIndexManager.RenewAIIndex()

		case "ClearOccupiedRoom":
			RoomReceptionist.CheckOccupiedRoom()

		case "Tester":
			authContent := Packet.AuthCMD{}
			if ok := authContent.Parse(packet.Content); !ok {
				log.Printf("Pare AuthCMD Failed.")
				return
			}
			if authContent.Token == "f2c884c50e8fb3544335c170e79f591e" {
				player.IsTester = true
				player.IsAuth = true
				err = Packet.SendPacket(player.Encoder, &Packet.Packet{
					Command:  "ReAuth",
					PacketID: packet.PacketID,
					Content: &Packet.ReAuthCMD{
						IsAuth: true,
					},
				})
				if err != nil {
					return
				}
			}

		case "Auth":
			authContent := Packet.AuthCMD{}
			if ok := authContent.Parse(packet.Content); !ok {
				log.Printf("Pare AuthCMD Failed.")
				return
			}
			_, err := fFunction.VerifyIDToken(authContent.Token)
			if err != nil {
				log.Printf("error verifying ID token: %v\n", err)
			} else {
				//log.Printf("Verified ID token: %v\n", token)
				player.IsAuth = true
				err = Packet.SendPacket(player.Encoder, &Packet.Packet{
					Command:  "ReAuth",
					PacketID: packet.PacketID,
					Content: &Packet.ReAuthCMD{
						IsAuth: true,
					},
				})
				if err != nil {
					return
				}
				continue
			}
			_ = Packet.SendPacket(player.Encoder, &Packet.Packet{
				Command:  "ReAuth",
				PacketID: packet.PacketID,
				ErrMsg:   err.Error(),
				Content: &Packet.ReAuthCMD{
					IsAuth: false,
				},
			})

		case "LeaveQuickRoom":
			if player.room != nil && !player.room.isStart && player.room.roomType == "Quick" {
				RoomReceptionist.PlayerLeaveRoom(player)
				player.LeaveRoom()
			} else {
				fmt.Println("LeaveRoom with Error Data: ", player)
			}

		case "CreateRoom":
			createRoomCMD := Packet.CreateRoomCMD{}
			if ok := createRoomCMD.Parse(packet.Content); !ok {
				log.Println("Parse CreateRoomCMD Failed.")
				return
			}
			roomGameDataSnap, ok := fFunction.GetRoomGameData(createRoomCMD.GameDataRoomUID)
			if !ok {
				if err = sendReCreateRoomWithErrStr(player, packet, "GAMEDATA_ROOMUID_GET_SNAP_ERROR"); err != nil {
					fmt.Println("sendReCreateRoomWithErrStr Failed when GAMEDATA_ROOMUID_GET_SNAP_ERROR: ", remoteAddr)
					return
				}
				continue
			}
			var roomGameData roomGameData
			err = roomGameDataSnap.DataTo(&roomGameData)
			if err != nil {
				fmt.Println("GAMEDATA_ROOMUID_DATA_TO_GAMEDATA_ERROR: ", remoteAddr)
				if err = sendReCreateRoomWithErrStr(player, packet, "GAMEDATA_ROOMUID_GET_GAMEDATA_ERROR"); err != nil {
					fmt.Println("sendReCreateRoomWithErrStr Failed when GAMEDATA_ROOMUID_GET_GAMEDATA_ERROR: ", remoteAddr)
					return
				}
				continue
			}

			fmt.Println("Player Enter Room. CheckRoomData: ", roomGameData)
			player.UID = createRoomCMD.MasterID
			if !player.IsTester {
				if betEnough, err := checkBetEnough(roomGameData.BetType, roomGameData.BetThreshold, player.UID); !betEnough {
					if err != nil {
						if sendERR := sendReCreateRoomWithErrStr(player, packet, "LOAD_BET_ERROR"); sendERR != nil {
							fmt.Println("sendReCreateRoomWithErrStr Failed when LOAD_BET_ERROR: ", player.UID, " addr: ", remoteAddr, "err:", err)
							return
						}
					} else {
						if err = sendReCreateRoomWithErrStr(player, packet, "BET_NOT_ENOUGH"); err != nil {
							fmt.Println("sendReCreateRoomWithErrStr Failed when BET_NOT_ENOUGH: ", player.UID, " addr: ", remoteAddr)
							return
						}
					}
					continue
				}
			}

			switch roomGameData.RoomType {
			case "Friend":
				//好友房預設須滿4人且不自動補人，只有測試環境才會跳過人數驗證 & 自動補人
				needCheckFull := true // Release版需要確認是否滿人，不給開未滿人的房
				autoFull := false
				if EnvironmentVersion == "Dev" || EnvironmentVersion == "Test" {
					needCheckFull = false
					autoFull = true
				}
				if EnvironmentVersion == "Release" || needCheckFull {
					autoFull = false
					isFullPlayer := true
					if len(createRoomCMD.PlayerIDs) != 4 {
						isFullPlayer = false
					}
					for i := 0; i < len(createRoomCMD.PlayerIDs); i++ {
						if createRoomCMD.PlayerIDs[i] == "" {
							isFullPlayer = false
						}
					}
					if !isFullPlayer {
						fmt.Println("CREATE_FRIEND_ROOM_WITH_NOT_FOUR_PLAYER: ", remoteAddr)
						if err = sendReCreateRoomWithErrStr(player, packet, "CREATE_FRIEND_ROOM_WITH_NOT_FOUR_PLAYER"); err != nil {
							fmt.Println("sendReCreateRoomWithErrStr Failed when CREATE_FRIEND_ROOM_WITH_NOT_FOUR_PLAYER: ", remoteAddr)
							return
						}
						continue
					}
				}
				player.WaitStr = roomGameData.RoomUID
				player.room = RoomReceptionist.PlayerJoinFriendRoom(player.WaitStr, roomGameData, &player, createRoomCMD.PlayerIDs, autoFull)
				if player.room == nil {
					fmt.Println("Player create friend room failed !? Para: ", player.WaitStr, roomGameData.RoomType, createRoomCMD.GameDataRoomUID, player, createRoomCMD.PlayerIDs, autoFull)
					if err = sendReCreateRoomWithErrStr(player, packet, "FRIEND_ROOM_CREATE_FAILED"); err != nil {
						fmt.Println("sendReCreateRoomWithErrStr Failed when FRIEND_ROOM_CREATE_FAILED: ", remoteAddr)
						return
					}
					continue
				}

			case "Quick":
				player.WaitStr = roomGameData.RoomUID
				player.room = RoomReceptionist.PlayerJoinQuickRoom(player.WaitStr, roomGameData, &player)
				if player.room == nil {
					fmt.Println("Player join quick room failed !? Para: ", player.WaitStr, roomGameData, player)
					if err = sendReCreateRoomWithErrStr(player, packet, "QUICK_ROOM_JOIN_FAILED"); err != nil {
						fmt.Println("sendReCreateRoomWithErrStr Failed when QUICK_ROOM_JOIN_FAILED: ", remoteAddr)
						return
					}
					continue
				}

			case "Guide":
				player.WaitStr = roomGameData.RoomUID
				player.room = RoomReceptionist.PlayerJoinGuideRoom(player.WaitStr, roomGameData, &player)
				if player.room == nil {
					fmt.Println("Player join guide room failed !? Para: ", player.WaitStr, roomGameData, player)
					if err = sendReCreateRoomWithErrStr(player, packet, "GUIDE_ROOM_JOIN_FAILED"); err != nil {
						fmt.Println("sendReCreateRoomWithErrStr Failed when GUIDE_ROOM_JOIN_FAILED: ", remoteAddr)
						return
					}
					continue
				}
			default:
				fmt.Println("GAMEDATA_ROOMTYPE_WRONG: ", remoteAddr)
				if err = sendReCreateRoomWithErrStr(player, packet, "GAMEDATA_ROOMTYPE_WRONG"); err != nil {
					fmt.Println("sendReCreateRoomWithErrStr Failed when GAMEDATA_ROOMTYPE_WRONG: ", remoteAddr)
					return
				}
				continue
			}

			player.room.checkStartAfterEnter(player.UID)

		default:
			return
		}
	}
}
