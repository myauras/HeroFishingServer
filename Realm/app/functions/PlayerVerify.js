exports = async function PlayerTokenVerify(data) {

  if (!("Token" in data) || !("Env" in data)) {
    console.log("[PlayerTokenVerify] 格式錯誤");
    return {
      Result: GameSetting.ResultTypes.Fail,
      Data: "格式錯誤",
    };
  }
  const ah = require("aurafortest-herofishing");

  // 使用 MongoDB Realm Admin API 可以參考官方文件: https://www.mongodb.com/docs/atlas/app-services/admin/api/v3/#section/Project-and-Application-IDs
  const adminApiUrl = `https://realm.mongodb.com/api/admin/v3.0/auth/providers/mongodb-cloud/login`;
  const apiPrivateKey = context.values.get("APIPrivateKey");
  const response = await context.http.post({
    url: adminApiUrl,
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: {
      apiKey: apiPrivateKey
    }
  });

  console.log("response: " + JSON.stringify(response))

  // 回傳結果
  if (response && response.status === 200) {
    const userId = JSON.parse(response.body.text()).user_id;
    let data = {
      playerID: userId,
    }
    return JSON.stringify(ah.ReplyData.NewReplyData(data, null));
  } else {
    let data = {
      playerID: null,
    }
    ah.WriteLog.Log(ah.GameSetting.LogType.PlayerVerify, null, "玩家token驗證失敗");
    return JSON.stringify(ah.ReplyData.NewReplyData(data, null));
  }
}
