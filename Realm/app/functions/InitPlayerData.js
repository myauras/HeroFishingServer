exports = async function InitPlayerData(data) {
  // if(context.user.id==""){
  //     console.log("context.user.id is empty")
  //     console.log(JSON.stringify(context.user))
  //     return 
  // }


  if (!("AuthType" in data)) {
    console.log("格式錯誤");
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
  result = await ah.DBManager.DB_InsertOne(ah.GameSetting.ColName.player, writePlayerDocData);
  if (!result) {
    console.log(`插入player文件錯誤 表格: ${ah.GameSetting.ColName.player}  文件: ${JSON.stringify(writePlayerDocData)}`);
    return JSON.stringify(ah.ReplyData.NewReplyData(null, "插入player表錯誤"));
  }

  return JSON.stringify(ah.ReplyData.NewReplyData(result, null));
}
