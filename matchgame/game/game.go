package game

import (
	"fmt"
	"log"
	"time"
)

const CHAN_BUFFER = 4

var EnvVersion string

var room GameRoom
var TIME_UPDATE_INTERVAL_MS float64

const AI_LOAD_STORAGE = true

type RoundEnum string

const (
	Round_Ba   RoundEnum = "Ba"   //一把
	Round_Quan RoundEnum = "Quan" //一圈
	Round_Jam  RoundEnum = "Jam"  //一將
)

type GameSetting struct {
	// GameData from firestore
	GameDataRoomUID      string  `firestore:"UID"`                  // GameDataRoomUID
	RoomType             string  `firestore:"RoomType"`             // (CustomProp) 好友房/快速配對房/教學房
	Bet                  string  `firestore:"Bet"`                  // (CustomProp) 金幣賭注
	BetType              string  `firestore:"BetType"`              // (CustomProp) 金鈔/紙鈔
	Invitable            bool    `firestore:"Invitable"`            // 可否邀請
	BallRate             float64 `firestore:"BallRate"`             // (CustomProp) 鋼珠倍數
	ThinkTime            float64 `firestore:"ThinkTime"`            // (CustomProp) 思考時間
	Round                string  `firestore:"Round"`                // (CustomProp) 幾把
	Enable               bool    `firestore:"Enable"`               // 是否開放
	Commission           float64 `firestore:"Commission"`           // 抽水
	PlayTileBalls        int     `firestore:"PlayTileBalls"`        // 打一張牌出幾顆鋼珠
	FullBagBalls         int     `firestore:"FullBagBalls"`         // 滿袋需要的小鋼珠數量
	FlowerBalls          int     `firestore:"FlowerBalls"`          // 打花牌時，小花鋼珠出的數量 (偶數) 目前小花鋼珠噴兩層所以要為偶數顯示才不會異常
	MC_FullBagEnergy     int     `firestore:"MC_FullBagEnergy"`     // 魔法陣能量滿需求多少能量
	MC_BallInEnergy      int     `firestore:"MC_BallInEnergy"`      // 每顆珠子進魔法陣增加的能量
	DiscardSeaAddSeconds int     `firestore:"DiscardSeaAddSeconds"` // 輪到自己摸牌時觀看次資訊可延長秒數

	// 也是 GameData form firestore，但server用不到
	// CreateTime
	// CronEvents
	// Priority
	// BGSprite
	// WaitAddBot
	// AddBotInterval
	// BetThreshold
	// GetOutThreshold
	// SheetBotPer

	// local para...
	RoomName       string                 // (房主UID+時間)toMD5
	RoomMasterUID  string                 // 房主UID
	GameServerIP   string                 //
	IsCheated      bool                   `firestore:"IsCheated"`
	CheatData      map[string]interface{} `firestore:"CheatData"` // (CustomProp) 作弊碼
	Ante           int                    //
	OnePointScore  int                    //
	GameServerPort int                    //
	MaxLeaveTime   float64                //
}

type WinOrder struct {
	IsAI         bool
	TotalScore   int
	PlayerUID    string
	OutputUID    string
	TotalWinBall int
}
type WinOrderList []WinOrder

type ErrorID string

const (
	ERROR_TOO_MANY_TILE        ErrorID = "TOO_MANY_CARDS"
	ERROR_STATE_STUCK          ErrorID = "SERVER_STUCK"
	ERROR_ADD_KONG_WITH_NIL    ErrorID = "ADD_KONG_WITH_NIL"
	ERROR_ACTION               ErrorID = "ACTION_ERROR"
	ERROR_TOO_MANY_TILE_MID_IN ErrorID = "TOO_MANY_TILE_MID_IN"
)

func (a WinOrderList) Len() int { // 重写 Len() 方法
	return len(a)
}
func (a WinOrderList) Swap(i, j int) { // 重写 Swap() 方法
	a[i], a[j] = a[j], a[i]
}
func (a WinOrderList) Less(i, j int) bool { // 重写 Less() 方法， 从大到小排序
	if a[j].TotalScore != a[i].TotalScore {
		return a[j].TotalScore < a[i].TotalScore
	}
	if a[i].PlayerUID == "" || a[j].PlayerUID == "" {
		if a[i].PlayerUID != "" {
			return true
		}
		if a[j].PlayerUID != "" {
			return false
		}
	}
	return false
}

func CheckOutCheatData(gameSetting *GameSetting) {
	if !gameSetting.IsCheated {
		gameSetting.CheatData = nil
	}
}

func InitGameRoom(firebaseDocID string, roomName string, playerIDs [PLAYER_NUMBER]string, outputPlayerUIDs [PLAYER_NUMBER]string, gameSetting GameSetting, waitRoom chan *GameRoom, serverName string) {
	if room.RoomName != "" {
		return
	}

	if TIME_UPDATE_INTERVAL_MS <= 0 {
		log.Println("Error Setting UDP Update interval.")
		TIME_UPDATE_INTERVAL_MS = 200
	}

	room.Init(gameSetting)
	room.SetDocumentID(firebaseDocID)
	room.SetRoomName(roomName)
	room.SetPlayers(playerIDs, outputPlayerUIDs)
	var logUIDs [PLAYER_NUMBER]string
	logUIDs = playerIDs
	if AI_LOAD_STORAGE {
		logUIDs = outputPlayerUIDs
	}

	createRoomLogData := map[string]interface{}{
		"UID":            roomName,
		"PlayerList":     logUIDs,
		"SettlementRoom": gameSetting.GameDataRoomUID,
		"CreateTime":     time.Now(),
		"Bet":            gameSetting.Bet,
		"ThinkTime":      gameSetting.ThinkTime,
		"ServerName":     serverName,
	}
	//FirebaseFunction.LogCreateGameRoom(createRoomLogData)
	FirebaseFunction.LogCreateGameRoomByRoomName(roomName, createRoomLogData)
	fmt.Println("InitGameRoom RoomType: ", gameSetting.RoomType)
	if gameSetting.RoomType == "Friend" {
		FirebaseFunction.AddCreateFriendRoomTimes(playerIDs[0])
	}
	waitRoom <- &room
}

func (g *GameRoom) WriteGameErrorLog(errorID ErrorID, data map[string]interface{}) {
	if data == nil {
		onlyErrorData := map[string]interface{}{
			"ErrorID": errorID,
		}
		docID, _ := FirebaseFunction.WriteErrorLog(onlyErrorData)
		g.ErrorLogs = append(g.ErrorLogs, docID)
	} else {
		data["ErrorID"] = errorID
		docID, _ := FirebaseFunction.WriteErrorLog(data)
		g.ErrorLogs = append(g.ErrorLogs, docID)
	}
}
