package setting

const (
	// 命名空間
	NAMESPACE_MATCHERSERVER = "herofishing-service"    // 配對伺服器命名空間
	NAMESPACE_GAMESERVER    = "herofishing-gameserver" // 遊戲伺服器命名空間

	// 服務名稱
	MATCHMAKER        = "herofishing-matchmaker"            // 配對伺服器Services名稱
	MATCHGAME         = "herofishing-matchgame"             // 遊戲房Services名稱
	MATCHGAME_TESTVER = "herofishing-matchgame-testver-tcp" // 個人測試用遊戲房Services名稱

	// 遊戲房舍定
	PLAYER_NUMBER = 4 // 遊戲房最多X位玩家
)

var EnvGCPProject = map[string]string{
	"Dev":     "fourth-waters-410202", // 開發版
	"Release": "herofishing-release",  // 正式版
}

// 環境版本
const (
	ENV_DEV     = "Dev"
	ENV_RELEASE = "Release"
)

// 配對類型結構
type MatchTypeStruct struct {
	Quick string // 快速配對
	Test  string // 測試房
}

// 配對類型
var MatchType = MatchTypeStruct{
	Quick: "Quick", // 快速配對
	Test:  "Test",  // 測試房
}
