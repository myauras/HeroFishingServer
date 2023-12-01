//剛進遊戲時執行，玩家遊玩中跨日(凌晨0點)也會執行
exports = async function Signin() {
  if (!context.user.id) {
    console.log("context.user.id is empty")
    console.log(JSON.stringify(context.user))
    return
  }

  const ah = require("aura-herofishing");

  // 如果玩家目前是下線的，改回上線中並更新最後登入時間
  let updateSuccess = await ah.DBManager.DB_UpdateOne(ah.GameSetting.ColName.player, {
    _id: context.user.id,
  }, {
    "$set": {
      onlineState: "Online",
      lastSigninAt: new Date()
    }
  }, null);
  if (!updateSuccess) {
    console.log("[Signin] 設為在線失敗")
    return JSON.stringify(ah.ReplyData.NewReplyData({}, "設為在線失敗"));
  }

  let now = new Date();

  // 更新在線時間
  updateSuccess = await ah.DBManager.DB_UpdateOne(ah.GameSetting.ColName.playerState, {
    _id: context.user.id
  }, {
    "$set": {
      lastUpdatedAt: now
    }
  }, null);

  if (!updateSuccess) {
    let error = "[Signin] 更新在線時間失敗";
    console.log(error);
    //寫Log
    ah.WriteLog.Log(ah.GameSetting.LogType.Signin, null, error);
    return JSON.stringify(ah.ReplyData.NewReplyData({}, "更新在線時間失敗"));
  }

  //寫Log
  ah.WriteLog.Log(ah.GameSetting.LogType.Signin, null, null);


  return JSON.stringify(ah.ReplyData.NewReplyData({}, null));
}
