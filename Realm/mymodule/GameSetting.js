module.exports = {
    EnvDB: Object.freeze({
        Dev: "herofishing",// 開發版
        Release: "???",// 正式版
    }),
    // DB集合
    ColName: Object.freeze({
        player: "player",
        playerCustom: "playerCustom",
        playerState: "playerState",
        playerHistory: "playerHistory",
        gameSetting: "gameSetting",
        gameLog: "gameLog",
        template: "template",
    }),
    // 環境版本對應Endpoint
    AppEndpoint: Object.freeze({
        Dev: "https://asia-south1.gcp.data.mongodb-api.com/app/aurafortest-bikmm",// 開發版
        Release: "???",// 正式版
    }),
    // 環境版本對應GroupID
    EnvGroupID: Object.freeze({
        Dev: "64e6b478a37b94153abe9042",// 開發版
        Release: "???",// 正式版
    }),
    // 環境版本對應AppID(AppID不是App的ObjectID)
    EnvAppID: Object.freeze({
        Dev: "aurafortest-bikmm",// 開發版
        Release: "???",// 正式版
    }),
    // 環境版本對應AppObjID
    EnvAppObjID: Object.freeze({
        Dev: "64e6d784c96a30ebafdf3de0",// 開發版
        Release: "???",// 正式版
    }),
    // 註冊類型
    AuthType: Object.freeze({
        Guest: "Guest",// 訪客
        Official: "Official",// 官方註冊
        Unknown: "Unknown",// 未知錯誤
    }),
    // 在線狀態
    OnlineState: Object.freeze({
        Online: "Online",// 在線
        Offline: "Offline",// 離線
    }),
    // 帳戶腳色(playerCustom中的腳色)
    PlayerCustomRole: Object.freeze({
        Player: "Player",// 玩家
        Developer: "Developer",// 開發者, 有更進階的DB訪問權限
    }),
    // 在線狀態
    LogType: Object.freeze({
        OnUserCreation: "OnUserCreation",// 玩家創Realm帳戶時會寫入此Log
        InitPlayerData: "InitPlayerData",// 玩家初始化玩家資料時會寫入此Log
        Signin: "Signin",// 玩家登入時寫入此Log
        PlayerVerify: "PlayerVerify",// 玩家登入時寫入此Log
    }),
    // 這邊要填入ColName的Key值, 如果template集合中有定義對應表的模板資料就要加在這裡
    ColTemplate: new Set(['player', 'playerState', 'playerHistory']),
}