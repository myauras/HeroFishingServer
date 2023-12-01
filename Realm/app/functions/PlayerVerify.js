exports = async function PlayerTokenVerify(data) {
  const ah = require("aura-herofishing");
  if (!("Token" in data) || !("Env" in data)) {
    console.log("[PlayerTokenVerify] 格式錯誤");
    return {
      Result: ah.GameSetting.ResultTypes.Fail,
      Data: "格式錯誤",
    };
  }

  // 取screet值 相關說明可以參考文件: https://www.mongodb.com/docs/atlas/app-services/values-and-secrets/#std-label-app-value
  const apiPublicKey = context.values.get("APIPublicKeyLink")
  const apiPrivateKey = context.values.get("APIPrivateKeyLink")


  // 使用 MongoDB Realm Admin API 可以參考官方文件: https://www.mongodb.com/docs/atlas/app-services/admin/api/v3/#section/Project-and-Application-IDs
  // 取得admin access_token
  const authEndpoint = 'https://realm.mongodb.com/api/admin/v3.0/auth/providers/mongodb-cloud/login';
  const authResponse = await context.http.post({
    url: authEndpoint,
    headers: {
      'Content-Type': ['application/json'],
      'Accept': ['application/json']
    },
    body: {
      username: apiPublicKey,
      apiKey: apiPrivateKey
    },
    encodeBodyAsJSON: true
  });
  // 取得admin access_token失敗
  if (!authResponse || authResponse.statusCode != 200) {
    let replyData = {
      playerID: null,
    }
    ah.WriteLog.Log(ah.GameSetting.LogType.PlayerVerify, verifyResponse, "取得admin access_token失敗");
    return JSON.stringify(ah.ReplyData.NewReplyData(replyData, null));
  }
  // 取得admin access_token成功
  let base64Data = authResponse.body.toBase64();
  let decodedText = new Buffer(base64Data, 'base64').toString('utf-8');
  let responseBody = JSON.parse(decodedText);
  const adminToken = responseBody.access_token;

  // 驗證token驗證
  const verifyEndpoint = `https://realm.mongodb.com/api/admin/v3.0/groups/${ah.GameSetting.EnvGroupID.Dev}/apps/${ah.GameSetting.EnvAppObjID.Dev}/users/verify_token`;
  const verifyResponse = await context.http.post({
    url: verifyEndpoint,
    headers: {
      'Accept': ['application/json'],
      'Authorization': [`Bearer ${adminToken}`]
    },
    body: {
      token: data.Token  // client access token
    },
    encodeBodyAsJSON: true
  });
  // 驗證token驗證
  if (!verifyResponse || verifyResponse.statusCode != 200) {
    let replyData = {
      playerID: null,
    }
    ah.WriteLog.Log(ah.GameSetting.LogType.PlayerVerify, verifyResponse, "玩家token驗證失敗");
    return JSON.stringify(ah.ReplyData.NewReplyData(replyData, null));
  }
  // 驗證token驗證成功
  // console.log("verifyResponse=" + JSON.stringify(verifyResponse));
  base64Data = verifyResponse.body.toBase64();
  decodedText = new Buffer(base64Data, 'base64').toString('utf-8');
  // console.log("decodedText=" + JSON.stringify(decodedText));
  responseBody = JSON.parse(decodedText);
  // console.log("responseBody: " + responseBody)
  const playerID = responseBody.custom_user_data._id;
  console.log("playerID=" + playerID);
  const userId = JSON.parse(verifyResponse.body.text()).user_id;
  let replyData = {
    playerID: userId,
  }

  return JSON.stringify(ah.ReplyData.NewReplyData(replyData, null));

  // // 使用Endpoint查找資料要先確保HTTPS Endpoints的Data API有開啟
  // const findEndpoint = ah.GameSetting.AppEndpoint.Dev + `endpoint/data/v1/action/findOne`; // https://asia-south1.gcp.data.mongodb-api.com/app/app-herofishing-pvxuj/endpoint/data/v1/action/findOne
  // console.log("verifyEndpoint=" + findEndpoint);
  // // 執行HTTP POST請求
  // const response = await context.http.post({
  //   url: findEndpoint,
  //   headers: {
  //     'Accept': ['application/json'],
  //     'Authorization': [`Bearer ${adminToken}`] 
  //   },
  //   body: {
  //     token: data.Token  // client access token
  //   },
  //   encodeBodyAsJSON: true
  // });
}
