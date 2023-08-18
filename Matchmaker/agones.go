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
	gameServerImage      = "gcr.io/majampachinko-develop/majampachinko-game-server:latest"
	gameserversNamespace = "default"
)

func CreateGameServer(roomName string, playerIDs []string, createrID string, roomID string, matchmakerPodName string) (*agonesv1.GameServer, error) {
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
		return nil, err
	}

	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
		return nil, err
	}

	// Create a GameServer
	myLabels := map[string]string{"roomName": roomName, "MasterID": createrID, "FrontendPodName": matchmakerPodName}
	myLabels["GameDataRoomUID"] = roomID
	for i := 0; i < len(playerIDs); i++ {
		myLabels[fmt.Sprintf("player%d", i)] = playerIDs[i]
	}

	allocacteInterface := agonesClient.AllocationV1().GameServerAllocations(gameserversNamespace)
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Required: allocationv1.GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"agones.dev/fleet": "simple-game-server"}}},
			MetaPatch: allocationv1.MetaPatch{
				Labels: myLabels},
		},
	}

	GameServerAllocation, err := allocacteInterface.Create(context.Background(), gsa, metav1.CreateOptions{})
	if err != nil {
		//panic(err)
		return nil, err
	}

	newGS, err := agonesClient.AgonesV1().GameServers(gameserversNamespace).Get(context.Background(), GameServerAllocation.Name, metav1.GetOptions{})
	fmt.Printf("New game servers' name is: %s, %s\n", newGS.ObjectMeta.Name, newGS.Status.Address)
	return newGS, err
}
