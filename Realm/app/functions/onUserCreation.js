/*
  1. 註冊新帳號時會自動執行這個function來建立Custom文件
  2. Custom文件會跟帳號綁定, 每次某個帳號登入時會自動取Custom文件(依據綁定的欄位資料來找文件), 並在送DB或Atlas Function等請求時會自動在請求中包含這份文件資料(所以此文件不能太大, 也不要塞沒必要的資料)
  3. 這份文件有些重要的用法, 其中最重要的就是DB在驗證這個使用者是否有權限訪問某個Collection或Document時, 可以用這份文件的欄位來驗證權限
  4. 此function會在帳戶建立後被呼叫 且此 function具有系統級別的訪問權
  5. 關於Custom User Data詳細資料可以參考官方文件: https://www.mongodb.com/docs/atlas/app-services/users/custom-metadata/
*/


exports = async function OnUserCreation(user) {
  const playerCustomCol = context.services
    .get("mongodb-atlas")
    .db("herofishing")
    .collection("playerCustom");

  const ah = require("aurafortest-herofishing");
  let role = ah.GameSetting.PlayerCustomRole.Player;
  // 建立玩家資料
  let writePlayerCustomData = {
    // 帳戶判斷綁定的是哪一份文件是依據, 用欄位id來綁定, 帳號登入會自動找playerCustom裡id符合帳戶id的文件作為custom data
    _id: user.id,
    // 紀錄建立時間
    createdAt: new Date(),
    // 腳色(一般玩家註冊帳戶的腳色都是Player, 用於控制讀寫DB的權限)
    role: role,
  };
  // 寫入Custom User Data資料
  let playerCustomDoc = await ah.DBManager.DB_InsertOne(ah.GameSetting.ColName.playerCustom, writePlayerCustomData);
  if (!playerCustomDoc) {
    let error = `[OnUserCreation] 寫入Custom User Data文件失敗, 帳戶id為: ${user.id}`;
    console.log(error);
    
    ah.WriteLog.Log(ah.GameSetting.LogType.OnUserCreation, null, error);
  } else {
    
    ah.WriteLog.Log(ah.GameSetting.LogType.OnUserCreation, playerCustomDoc, null);
  }


}