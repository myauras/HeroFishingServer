module.exports = {
    // DB集合
    ColName: Object.freeze({
        player: "player",
        playerState: "playerState",
        playerHistory: "playerHistory",
        gameSetting: "gameSetting",
        gameLog: "gameLog",
        template: "template",
    }),
    // 註冊類型
    AuthType: Object.freeze({
        Guest: "Guest",
        Official: "Official",
        Unknown: "Unknown",
    }),
    // 在線狀態
    OnlineState: Object.freeze({
        Online: "Online",
        Offline: "Offline",
    }),
    // 這邊要填入ColName的Key值
    ColTemplate: new Set(['player', 'playerState', 'playerHistory']),
}