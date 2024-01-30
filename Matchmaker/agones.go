package main

import (
	"context"
	"fmt"
	logger "matchmaker/logger"
	"strconv"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	gameserverName       = "herofishing-matchgame"
	gameserversNamespace = "herofishing-gameserver"
)

var agonesClient *versioned.Clientset

// 初始化Agones Client
func InitAgones() error {
	log.Infof("%s 開始初始化Agones API Client", logger.LOG_Agones)
	var err error
	// 取目前pod所在k8s cluster的config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("%s 取cluster的config錯誤: %v", logger.LOG_Agones, err)
		return err
	}
	// 與agones連接
	agonesClient, err = versioned.NewForConfig(config)
	if err != nil {
		log.Errorf("%s 建立 agones api client錯誤: %v", logger.LOG_Agones, err)
		return err
	}
	log.Infof("%s 初始化Agones API Client完成", logger.LOG_Agones)
	return nil
}

// 取得Agones API Client
func GetAgonesClient() *versioned.Clientset {
	if agonesClient == nil {
		log.Infof("%s 尚未建立 agones api client, 嘗試建立", logger.LOG_Agones)
		err := InitAgones()
		if err != nil {
			return nil
		}
	}
	return agonesClient
}

// 建立一個新的Matchgame Server
func CreateGameServer(packID int, roomName string, playerIDs []string, createrID string, dbMapID string, matchmakerPodName string) (*agonesv1.GameServer, error) {

	// 建立遊戲房伺服器標籤
	myLabels := map[string]string{
		"PackID":            strconv.Itoa(packID), // 需要傳入packID, 等房間創好後會pub通知matchmaker房間創好了, 此時matchmaker才使用此packID回送client通知可以連線matchgame了
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

// 檢查該Matchgame server是正常運作
func CheckGameServer(roomName string) error {
	gameServers, err := agonesClient.AgonesV1().GameServers(gameserversNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("取得Matchgame列表失敗: %v", err)
		return err
	}
	log.Infof("檢查RoomName是否可加入: %s", roomName)
	for _, gs := range gameServers.Items {
		log.Infof("GS RoomName: %s", gs.Labels["RoomName"])
		if gs.Labels["RoomName"] == roomName {
			log.Infof("目標RoomName狀態: %s", gs.Status.State)
			if gs.Status.State == agonesv1.GameServerStateReady {
				log.Infof("%s 已確認目標Matchgame server正常運作", logger.LOG_Agones)
				return nil
			} else {
				return fmt.Errorf("%s Matchgame(%s)掛了", logger.LOG_Agones, roomName)
			}
		}
	}
	return fmt.Errorf("%s Matchgame(%s)不存在", logger.LOG_Agones, roomName)
}
