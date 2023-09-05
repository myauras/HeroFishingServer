exports = async function InitPlayerData(data) {
  if (!context.user.id) {
    console.log("context.user.id is empty")
    console.log(JSON.stringify(context.user))
    return
  }


  if (!("AuthType" in data)) {
    console.log("[InitPlayerData] 格式錯誤");
    return {
      Result: GameSetting.ResultTypes.Fail,
      Data: "格式錯誤",
    };
  }
  const ah = require("aurafortest-herofishing");
  // 建立玩家資料
  writePlayerDocData = {
    "_id": context.user.id,
    "authType": data.AuthType,
    "point": 100,
    "onlineState": ah.GameSetting.OnlineState.Online,
  };
  // 寫入玩家資料
  playerDoc = await ah.DBManager.DB_InsertOne(ah.GameSetting.ColName.player, writePlayerDocData);
  if (!playerDoc) {
    let error = `[InitPlayerData] 插入player文件錯誤 表格: ${ah.GameSetting.ColName.player}  文件: ${JSON.stringify(writePlayerDocData)}`;
    console.log(error);
    //寫Log
    ah.WriteLog.Log(ah.GameSetting.LogType.InitPlayerData, null, error);
    return JSON.stringify(ah.ReplyData.NewReplyData(null, "插入player表錯誤"));
  }

  //寫Log
  ah.WriteLog.Log(ah.GameSetting.LogType.InitPlayerData, playerDoc, null);


  return JSON.stringify(ah.ReplyData.NewReplyData(playerDoc, null));
}
