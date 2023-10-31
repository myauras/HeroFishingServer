exports = async function SubTest() {
  const ah = require("aurafortest-herofishing");
  console.log("SubTest2")
  let myJson = {};
  try {
    myJson = await ah.GameJson.Get(ah.GameSetting.JsonName.Hero);
    console.log(JSON.stringify(myJson))
  } catch (error) {
    console.error("Error during getting Hero JSON:", error);
  }
  return JSON.stringify(ah.ReplyData.NewReplyData(myJson, null));

}
