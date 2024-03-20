package setting

import (
	"encoding/json"
	"net"
	"sync"
)

// 伺服器設定
const (
	GAMEUPDATE_MS                  = 1000  // 每X毫秒送UPDATEGAME_TOCLIENT封包給client(遊戲狀態更新並心跳檢測)
	PLAYERUPDATE_MS                = 1000  // 每X毫秒送UPDATEPLAYER_TOCLIENT封包給client(玩家狀態更新)
	SCENEUPDATE_MS                 = 10000 // 每X毫秒送UPDATESCENE_TOCLIENT封包給client(場景狀態更新)
	ROOMLOOP_MS                    = 100   // 每X毫秒房間檢查一次
	AGONES_HEALTH_PIN_INTERVAL_SEC = 1     // 每X秒檢查AgonesServer是否正常運作(官方文件範例是用2秒)
	TCP_CONN_TIMEOUT_SEC           = 120   // TCP連線逾時時間X秒
	BOT_TARGET_MS                  = 1000  // Bot沒目標時, 每X豪秒選一次目標
	BOT_CHANGE_TARGET_MS           = "10000~20000"  // Bot主動更換目標間隔X豪秒區間
	BOT_ATTACK_MS                  = 300  // Bot有目標時, 攻擊頻率為X豪秒
)

type ConnectionTCP struct {
	Conn       net.Conn      // TCP連線
	Encoder    *json.Encoder // 連線編碼
	Decoder    *json.Decoder // 連線解碼
	MyLoopChan *LoopChan
}

// 關閉PackReadStopChan通道
func (loopChan *LoopChan) ClosePackReadStopChan() {
	loopChan.ChanCloseOnce.Do(func() {
		close(loopChan.StopChan)
	})
}

type LoopChan struct {
	StopChan      chan struct{} // 讀取封包Chan
	ChanCloseOnce sync.Once
}

type ConnectionUDP struct {
	Conn       net.PacketConn // UDP連線
	Addr       net.Addr       // 玩家連線地址
	ConnToken  string         // 驗證Token
	MyLoopChan *LoopChan
}
