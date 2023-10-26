package setting

const (
	// 命名空間
	NAMESPACE_MATCHERSERVER = "herofishing-matchserver" // 配對伺服器命名空間
	NAMESPACE_GAMESERVER    = "herofishing-gameserver"  // 遊戲伺服器命名空間

	// 服務名稱
	MATCHMAKER = "herofishing-matchmaker" // 配對伺服器名稱
	MATCHGAME  = "herofishing-matchgame"  // 遊戲房名稱

	// 遊戲房舍定
	PLAYER_NUMBER = 4 // 遊戲房最多X位玩家
)

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
