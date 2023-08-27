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


  const cluster = context.services.get("Cluster0");
  const db = cluster.db("herofishing");
  const playerCol = db.collection("player");
  await playerCol.insertOne({
    "_id": context.user.id,

  });

  
  const manager = require("aurafortest-herofishing")
  return JSON.stringify(manager.NewData("A","b"))
}
