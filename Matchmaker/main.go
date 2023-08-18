package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

var EvnVersion string             // 環境版本(Dev, Release)
var SelfPodName string            // K8s上所屬的Pod名稱
var Receptionist RoomReceptionist // 房間接待員

func main() {
	log.Infof("%s ==============MATCHMAKER START==============", LOG_Main)

	// 設定Port
	port := flag.String("port", "32680", "The port to listen to tcp traffic on")
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	log.Infof("%s Port: %s", LOG_Main, *port)

	// 設定環境版本
	EvnVersion = *flag.String("Version", "Dev", "EnvVersion setting")
	if ep := os.Getenv("Version"); ep != "" {
		EvnVersion = ep
	}
	log.Infof("%s EvnVersion: %s", LOG_Main, EvnVersion)

	// 設定K8s上所屬的Pod名稱
	SelfPodName = *flag.String("MY_POD_NAME", "myPodName", "Pod Name")
	if ep := os.Getenv("MY_POD_NAME"); ep != "" {
		SelfPodName = ep
	}

	// 偵聽TCP封包
	src := ":" + *port
	listener, err := net.Listen("tcp", src)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer listener.Close()
	log.Infof("%s TCP server start and listening on %s.\n", LOG_Main, src)

	Receptionist.Init()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Infof("%s Connection error %s.\n", LOG_Main, err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Infof("%s Client connected from: %s", LOG_Main, remoteAddr)
	defer conn.Close()

	// player := roomPlayer{
	// 	id:      "",
	// 	isAuth:  false,
	// 	conn:    conn,
	// 	encoder: json.NewEncoder(conn),
	// 	decoder: json.NewDecoder(conn),
	// 	mapID:   "",
	// 	room:    nil,
	// }
}
