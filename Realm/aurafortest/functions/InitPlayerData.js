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

  // 建立玩家資料
  writePlayerDocData={
    "_id":context.user.id,
    "authType": data.AuthType,
    "point":Int64(0),
    "onlineState":gm.GameSetting.OnlineState.Online,
  };
  // 寫入玩家資料
  await gm.DBManager.DB_InsertOne(gm.GameSetting.ColName.Player,writePlayerDocData)

  


  return JSON.stringify(gm.ReplyData.NewReplyData(null,null))
}
