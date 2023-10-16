package main

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	logger "matchmaker/logger"
)

const (
	gameserverName       = "herofishing-matchgame"
	gameserversNamespace = "herofishing-gameserver"
)

func CreateGameServer(roomName string, playerIDs []string, createrID string, mapID string, matchmakerPodName string) (*agonesv1.GameServer, error) {
	// 取目前pod所在k8s cluster的config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("%s Could not create in cluster config: %v", logger.LOG_Agones, err)
		return nil, err
	}

	// 與agones連接
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		log.Errorf("%s Could not create the agones api clientset: %v", logger.LOG_Agones, err)
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

	for key, value := range myLabels {
		log.Infof("%s label Key&Value   %s : %s", logger.LOG_Agones, key, value)
	}

	// 分配game server
	allocacteInterface := agonesClient.AllocationV1().GameServerAllocations(gameserversNamespace)
	// 定義規範- 找game server(pod)並新增標籤
	gsAllocation := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			// 找fleet.yaml定義的fleet metadata名稱
			Required: allocationv1.GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"agones.dev/fleet": gameserverName}}},
			// 在產生的pod上新增Label
			MetaPatch: allocationv1.MetaPatch{
				Labels: myLabels},
		},
	}
	// 使用規範來建立game server(pod)並新增標籤
	GameServerAllocation, err := allocacteInterface.Create(context.Background(), gsAllocation, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("%s 建立game server失敗: %v", logger.LOG_Agones, err)
		return nil, err
	}

	newGS, err := agonesClient.AgonesV1().GameServers(gameserversNamespace).Get(context.Background(), GameServerAllocation.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("%s 取得game server失敗: %v", logger.LOG_Agones, err)
		return nil, err
	}

	log.Infof("%s New game servers name: %s    address: %s\n", logger.LOG_Agones, newGS.ObjectMeta.Name, newGS.Status.Address)
	return newGS, err
}
