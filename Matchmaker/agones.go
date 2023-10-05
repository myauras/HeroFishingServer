package main

import (
	"context"
	"fmt"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	gameServerImage      = "asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchmaker:latest"
	gameserversNamespace = "default"
)

func CreateGameServer(roomName string, playerIDs []string, createrID string, mapID string, matchmakerPodName string) (*agonesv1.GameServer, error) {
	// 取目前pod所在k8s cluster的config
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
		return nil, err
	}

	// 與agones連接
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
		return nil, err
	}

	// 建立遊戲房伺服器標籤
	myLabels := map[string]string{
		"RoomName":          roomName,
		"CreaterID":         createrID,
		"MatchmakerPodName": matchmakerPodName,
		"MapID":             mapID,
	}

	for i := 0; i < len(playerIDs); i++ {
		myLabels[fmt.Sprintf("player%d", i)] = playerIDs[i]
	}

	// 分配game server
	allocacteInterface := agonesClient.AllocationV1().GameServerAllocations(gameserversNamespace)
	// 定義規範- 找game server(pod)並新增標籤
	gsAllocation := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			// 找fleet.yaml定義的fleet metadata名稱
			Required: allocationv1.GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"agones.dev/fleet": "herofishing-matchgame"}}},
			// 在產生的pod上新增Label
			MetaPatch: allocationv1.MetaPatch{
				Labels: myLabels},
		},
	}
	// 使用規範來建立game server(pod)並新增標籤
	GameServerAllocation, err := allocacteInterface.Create(context.Background(), gsAllocation, metav1.CreateOptions{})
	if err != nil {
		//panic(err)
		return nil, err
	}

	newGS, err := agonesClient.AgonesV1().GameServers(gameserversNamespace).Get(context.Background(), GameServerAllocation.Name, metav1.GetOptions{})
	fmt.Printf("New game servers' name is: %s, %s\n", newGS.ObjectMeta.Name, newGS.Status.Address)
	return newGS, err
}
