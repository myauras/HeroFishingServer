exports = async function PlayerTokenVerify(data) {
  const ah = require("aurafortest-herofishing");
  if (!("Token" in data) || !("Env" in data)) {
    console.log("[PlayerTokenVerify] 格式錯誤");
    return {
      Result: ah.GameSetting.ResultTypes.Fail,
      Data: "格式錯誤",
    };
  }

  // 使用 MongoDB Realm Admin API 可以參考官方文件: https://www.mongodb.com/docs/atlas/app-services/admin/api/v3/#section/Project-and-Application-IDs
  const adminApiUrl = `https://realm.mongodb.com/api/admin/v3.0/auth/providers/mongodb-cloud/login`;
  // 取screet值 相關說明可以參考文件: https://www.mongodb.com/docs/atlas/app-services/values-and-secrets/#std-label-app-value
  const apiPrivateKey = context.values.get("APIPrivateKeyLink");
  context.values.get("adminUsers");
  console.log("apiPrivateKey=" + apiPrivateKey);
  const response = await context.http.post({
    url: adminApiUrl,
    headers: {
      'Authorization': [`Bearer ${data.Token}`],
      'Content-Type': ['application/json']
    },
    body: {
      apiKey: apiPrivateKey
    },
    encodeBodyAsJSON: true
  });

  console.log("response: " + JSON.stringify(response))

  // 驗證失敗
  if (!response || response.statusCode != 200) {
    let replyData = {
      playerID: null,
    }
    ah.WriteLog.Log(ah.GameSetting.LogType.PlayerVerify, response, "玩家token驗證失敗");
    return JSON.stringify(ah.ReplyData.NewReplyData(replyData, null));
  }

  //驗證成功
  const userId = JSON.parse(response.body.text()).user_id;
  let replyData = {
    playerID: userId,
  }
  return JSON.stringify(ah.ReplyData.NewReplyData(replyData, null));
}
