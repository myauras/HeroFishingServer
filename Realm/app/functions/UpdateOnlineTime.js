exports = async function UpdateOnlineTime() {
  if (!context.user.id) {
    console.log("context.user.id is empty")
    console.log(JSON.stringify(context.user))
    return
  }

  const ah = require("aurafortest-herofishing");

  // 如果玩家目前是下線的，改回上線中並更新最後登入時間
  await ah.DBManager.DB_UpdateOne(ah.GameSetting.ColName.player, {
    _id: context.user.id,
    onlineState: "Offline"
  }, {
    "$set": {
      onlineState: "Online",
      lastSigninAt: new Date()
    }
  }, null);


  // 更新在線時間
  let updateSuccess = await ah.DBManager.DB_UpdateOne(ah.GameSetting.ColName.playerState, {
    _id: context.user.id
  }, {
    "$set": {
      lastUpdateAt: new Date()
    }
  }, null);

  if (!updateSuccess) {
    console.log("[InitPlayerData] 更新在線時間失敗");
    return JSON.stringify(ah.ReplyData.NewReplyData(null, "更新在線時間失敗"));
  }


  return JSON.stringify(ah.ReplyData.NewReplyData(null, null));

}
