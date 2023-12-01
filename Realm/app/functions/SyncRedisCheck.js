// 如果玩家的redisSync為false代表之前遊戲結束沒有結算好資料, 要把RedisDB的資料寫回player
exports = async function SyncRedisCheck() {
  if (!context.user.id) {
    console.log("context.user.id is empty")
    console.log(JSON.stringify(context.user))
    return
  }

  const ah = require("aura-herofishing");
  let playerDoc = await ah.DBManager.DB_FindOne(ah.GameSetting.ColName.player, { _id: context.user.id }, { redisSync: 1 });
  if (!playerDoc)
    return JSON.stringify(ah.ReplyData.NewReplyData({}, null));
  if (playerDoc.redisSync)
    return JSON.stringify(ah.ReplyData.NewReplyData({}, null));
  let redisPlayerDoc = await ah.RedisDBManager.FindOne(context.user.id);
  if (!redisPlayerDoc)
    return JSON.stringify(ah.ReplyData.NewReplyData({}, null));

  console.log(`玩家 ${context.user.id} 需同步redisDB資料`);
  await ah.DBManager.DB_UpdateOne(ah.GameSetting.ColName.player, { _id: context.user.id }, {

    point: redisPlayerDoc.point,
    heroExp: redisPlayerDoc.heroExp,
    redisSync: true,
  }, null)
  console.log(`玩家 ${context.user.id} redisDB資料同步完成`);

  return JSON.stringify(ah.ReplyData.NewReplyData({}, null));

}
