exports = async function GetServerTime() {
  if (!context.user.id) {
    console.log("context.user.id is empty")
    console.log(JSON.stringify(context.user))
    return
  }

  const ah = require("aura-herofishing");

  let data = {
    serverTime: new Date(),
  }

  return JSON.stringify(ah.ReplyData.NewReplyData(data, null));
}
