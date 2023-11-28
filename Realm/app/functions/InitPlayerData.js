exports = async function InitPlayerData(data) {
  if (!context.user.id) {
    console.log("context.user.id is empty")
    console.log(JSON.stringify(context.user))
    return
  }
  const ah = require("aurafortest-herofishing");

  if (!("AuthType" in data)) {
    console.log("[InitPlayerData] 格式錯誤");
    return {
      Result: ah.GameSetting.ResultTypes.Fail,
      Data: "格式錯誤",
    };
  }

  // 建立plyer資料
  writePlayerDocData = {
    _id: context.user.id,
    authType: data.AuthType,
    point: 100,
    onlineState: ah.GameSetting.OnlineState.Online,
  };
  // 寫入plyer資料
  let playerDoc = await ah.DBManager.DB_InsertOne(ah.GameSetting.ColName.player, writePlayerDocData);
  if (!playerDoc) {
    let error = `[InitPlayerData] 插入player文件錯誤 表格: ${ah.GameSetting.ColName.player}  文件: ${JSON.stringify(writePlayerDocData)}`;
    console.log(error);
    //寫Log
    ah.WriteLog.Log(ah.GameSetting.LogType.InitPlayerData, null, error);
    return JSON.stringify(ah.ReplyData.NewReplyData({}, "插入player表錯誤"));
  }

  // 建立playerState資料
  writePlayerStateDocData = {
    _id: context.user.id,
    lastUpdatedAt: new Date(),
    inMatchgameID: "",
  };
  // 寫入playerState資料
  let playerStateDoc = await ah.DBManager.DB_InsertOne(ah.GameSetting.ColName.playerState, writePlayerStateDocData);
  if (!playerStateDoc) {
    let error = `[InitPlayerData] 插入playerState文件錯誤 表格: ${ah.GameSetting.ColName.playerState}  文件: ${JSON.stringify(writePlayerStateDocData)}`;
    console.log(error);
    return JSON.stringify(ah.ReplyData.NewReplyData({}, "插入playerState表錯誤"));
  }

  //寫Log
  ah.WriteLog.Log(ah.GameSetting.LogType.InitPlayerData, playerDoc, null);


  return JSON.stringify(ah.ReplyData.NewReplyData(playerDoc, null));
}
