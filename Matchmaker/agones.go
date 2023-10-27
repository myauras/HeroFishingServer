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

func CreateGameServer(roomName string, playerIDs []string, createrID string, dbMapID string, matchmakerPodName string) (*agonesv1.GameServer, error) {
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
		"DBMapID":           dbMapID,
	}

	for i := 0; i < len(playerIDs); i++ {
		key := fmt.Sprintf("Player%d", i)
		myLabels[key] = playerIDs[i]
	}

	for key, value := range myLabels {
		log.Infof("%s label Key&Value   %s : %s", logger.LOG_Agones, key, value)
	}

	// 分配game server
	allocacteInterface := agonesClient.AllocationV1().GameServerAllocations(gameserversNamespace)
	// 定義規範- 找game server(pod)並新增標籤
	gsAllocationName := fmt.Sprintf("%s_allocation", matchmakerPodName)
	gsAllocation := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gsAllocationName,     // 資源的唯一名稱
			Namespace: gameserversNamespace, // 通常是 "default" 或你自己定義的
		},
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
	//log.Infof("%s Preparing to create game server with labels: %+v", logger.LOG_Agones, gsAllocation.Spec.MetaPatch.Labels)
	log.Infof("gsAllocation: %+v", gsAllocation)

	GameServerAllocation, err := allocacteInterface.Create(context.Background(), gsAllocation, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("%s 建立game server失敗: %v", logger.LOG_Agones, err)
		return nil, err
	} else {
		log.Infof("%s 建立game server成功", logger.LOG_Agones)
	}

	log.Infof("%s Allocation State: %v", logger.LOG_Agones, GameServerAllocation.Status.State)
	newGSName := GameServerAllocation.Status.GameServerName
	log.Infof("%s newGSName: %v", logger.LOG_Agones, newGSName)
	newGS, err := agonesClient.AgonesV1().GameServers(gameserversNamespace).Get(context.Background(), newGSName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("%s 取得game server失敗: %v", logger.LOG_Agones, err)
		return nil, err
	} else {
		log.Infof("%s 取得game server成功", logger.LOG_Agones)
	}
	log.Infof("%s New game servers name: %s    address: %s   port: %v", logger.LOG_Agones, newGS.ObjectMeta.Name, newGS.Status.Address, newGS.Status.Ports[0].Port)
	return newGS, err

}
