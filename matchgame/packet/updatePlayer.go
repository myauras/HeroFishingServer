package packet

type UpdatePlayer_ToClient struct {
	CMDContent
	Players [4]*Player // 玩家陣列

}
type Player struct {
	ID          string
	Idx         int          // 玩家索引(座位)
	GainPoints  int64        // 玩家總獲得點數
	PlayerBuffs []PlayerBuff // 玩家Buffer
}
type PlayerBuff struct {
	Name     string  // 效果名稱
	Value    float64 // 效果數值
	AtTime   float64 // 在遊戲時間第X秒觸發
	Duration float64 // 效果持續X秒
}
