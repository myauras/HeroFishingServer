// 這邊有設定新的Collection Template也要在DBManager那邊設定
let db = db.getSiblingDB('herofishing');

// 刪除整個template collection
db.template.deleteMany({});

let nowDate = new Date();
// 開始插入模板
let playerDoc = db.template.insertMany([
  // 模板-玩家資料
  {
    _id: "player",
    createdAt: nowDate,
    authType: "Guest",
    point: 1,//NumberLong("1")
    onlineState: "Offline",
    lastSigninAt_nowDate: null,
    lastSignoutAt_nowDate: null,
    ban: false,
    deviceUID: "",
  },
  // 模板-玩家狀態
  {
    _id: "playerState",
    createdAt: nowDate,
    lastUpdateAt_nowDate: null,
  },
  // 模板-玩家歷程
  {
    _id: "playerHistory",
    createdAt: nowDate,
  }
]);

printjson(playerDoc);