package mongo

import (
	"herofishingGoModule/setting"
	"time"
)

var (
	Env           = "Dev" // 目前的環境版本，初始化時會設定
	APIPublicKey  = ""    // 目前的Realm的APIKey，初始化時會設定
	APIPrivateKey = ""    // 目前的Realm的APIKey，初始化時會設定
)

const ()

var EnvDBUri = map[string]string{
	"Dev":     "mongodb+srv://%s:%s@cluster-herofishing.8yp6fou.mongodb.net/?retryWrites=true&w=majority", // 開發版
	"Release": "???",                                                                                      // 正式版
}

var AppEndpoint = map[string]string{
	"Dev":     "https://asia-south1.gcp.data.mongodb-api.com/app/app-herofishing-pvxuj", // 開發版
	"Release": "???",                                                                    // 正式版
}

// GroupID就是ProjectID(在atlas app service左上方有垂直三個點那點Project Settings)
// 也可以在開啟Atlas Services時 網址會顯示ProjectID
// 在https://realm.mongodb.com/groups/653cd1ccb544ec4945f8df83/apps/653cd937e285e8ddc4d6ac57/dashboard中
// https://realm.mongodb.com/groups/[GroupID]/apps/[App ObjectID]/dashboard
var EnvGroupID = map[string]string{
	"Dev":     "653cd1ccb544ec4945f8df83", // 開發版
	"Release": "???",                      // 正式版
}

// AppID
var EnvAppID = map[string]string{
	"Dev":     "app-herofishing-pvxuj", // 開發版
	"Release": "???",                   // 正式版
}

// App ObjectID跟AppID不一樣, 開啟Atlas Services時 網址會顯示App ObjectID
// https://realm.mongodb.com/groups/653cd1ccb544ec4945f8df83/apps/653cd937e285e8ddc4d6ac57/dashboard
// https://realm.mongodb.com/groups/[GroupID]/apps/[App ObjectID]/dashboard
var EnvAppObjID = map[string]string{
	"Dev":     "653cd937e285e8ddc4d6ac57", // 開發版
	"Release": "???",                      // 正式版
}

var EnvDB = map[string]string{
	"Dev": "herofishing", // 開發版
}

const (
	MATCH_QUICK = "Quick"
)

// Collection名稱列表結構
type ColNameStruct struct {
	// 遊戲設定
	GameSetting string
	GameLog     string
	Template    string
	Map         string

	// 玩家資料
	Player        string
	PlayerCustom  string
	PlayerState   string
	PlayerHistory string

	// 遊戲資料
	Matchgame string
}

// Collection名稱列表
var ColName = ColNameStruct{

	// 遊戲設定
	GameSetting: "gameSetting",
	GameLog:     "gameLog",
	Template:    "template",
	Map:         "map",

	// 玩家資料
	Player:        "player",
	PlayerCustom:  "playerCustom",
	PlayerState:   "playerState",
	PlayerHistory: "playerHistory",

	// 遊戲資料
	Matchgame: "matchgame",
}

// DB地圖資料
type DBMap struct {
	ID           string `bson:"_id"`
	MatchType    string `bson:"matchType"`
	JsonMapID    int32  `bson:"jsonMapID"`
	Bet          int32  `bson:"bet"`
	BetThreshold int64  `bson:"betThreshold"`
	Enable       bool   `bson:"enable"`
}

// 遊戲房資料
type DBMatchgame struct {
	ID        string    `bson:"_id"`
	CreatedAt time.Time `bson:"createdAt"`
	DBMapID   string    `bson:"dbMapID"`
	// 玩家陣列(索引0~3 分別代表4個玩家)
	// 1. 索引代表玩家座位
	// 2. 座位無關玩家進來順序 有人離開就會空著 例如 索引2的玩家離開 players[2]就會是nil 直到有新玩家加入
	PlayerIDs         [setting.PLAYER_NUMBER]string `bson:"playerIDs"`
	IP                string                        `bson:"ip"`
	Port              int32                         `bson:"port"`
	NodeName          string                        `bson:"nodeName"`
	PodName           string                        `bson:"podName"`
	MatchmakerPodName string                        `bson:"matchmakerPodName"`
}
