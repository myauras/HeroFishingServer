package mongo

import "time"

var (
	Env           = "Dev"                                  // 環境版本，初始化時會設定
	APIPublicKey  = "rqjseyja"                             // Realm的APIKey，初始化時會設定
	APIPrivateKey = "e8cc1224-f04b-46c7-aee0-19309dc499dc" // Realm的APIKey，初始化時會設定
)

const ()

var EnvDBUri = map[string]string{
	"Dev":     "mongodb+srv://%s:%s@cluster0.edk0n6b.mongodb.net/?retryWrites=true&w=majority", // 開發版
	"Release": "???",                                                                           // 正式版
}

var AppEndpoint = map[string]string{
	"Dev":     "https://asia-south1.gcp.data.mongodb-api.com/app/aurafortest-bikmm", // 開發版
	"Release": "???",                                                                // 正式版
}

var EnvGroupID = map[string]string{
	"Dev":     "64e6b478a37b94153abe9042", // 開發版
	"Release": "???",                      // 正式版
}

var EnvAppID = map[string]string{
	"Dev":     "aurafortest-bikmm", // 開發版
	"Release": "???",               // 正式版
}

var EnvAppObjID = map[string]string{
	"Dev":     "64e6d784c96a30ebafdf3de0", // 開發版
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
	JsonMapID    string `bson:"jsonMapID"`
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
	PlayerIDs         []string `bson:"playerIDs"`
	IP                string   `bson:"ip"`
	Port              int32    `bson:"port"`
	NodeName          string   `bson:"nodeName"`
	PodName           string   `bson:"podName"`
	MatchmakerPodName string   `bson:"matchmakerPodName"`
}
