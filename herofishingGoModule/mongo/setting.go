package mongo

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

// Collection名稱列表結構
type ColNameStruct struct {
	Player        string
	PlayerCustom  string
	PlayerState   string
	PlayerHistory string
	GameSetting   string
	GameLog       string
	Template      string
}

// Collection名稱列表
var ColName = ColNameStruct{
	Player:        "player",
	PlayerCustom:  "playerCustom",
	PlayerState:   "playerState",
	PlayerHistory: "playerHistory",
	GameSetting:   "gameSetting",
	GameLog:       "gameLog",
	Template:      "template",
}
