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

  const gm = require("aurafortest-herofishing")
  const mongodb = require("mongodb");

  // 建立玩家資料
  writePlayerDocData = {
    "_id": context.user.id,
    "authType": data.AuthType,
    "point": mongodb.NumberLong("100"),
    "onlineState": gm.GameSetting.OnlineState.Online,
  };
  // 寫入玩家資料
  result = await gm.DBManager.DB_InsertOne(gm.GameSetting.ColName.Player, writePlayerDocData);
  if (!result) {
    console.log("插入player表錯誤");
    return JSON.stringify(gm.ReplyData.NewReplyData(null, "插入player表錯誤"));
  }

  return JSON.stringify(gm.ReplyData.NewReplyData(result, null));
}
