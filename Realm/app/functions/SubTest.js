exports = async function SubTest() {

  const ah = require("aurafortest-herofishing");
  // 取Cloud Pub/Sub 的JSON
  const { PubSub } = require('@google-cloud/pubsub');
  // 創建 PubSub 客戶端
  const pubsub = new PubSub();
  const topicName = 'herofishing-json-topic';
  const topic = pubsub.topic(topicName);

  const [messages] = await topic.get({ maxResults: 1 });
  const jsonData = {}
  if (messages.length > 0) {
    jsonData = JSON.parse(messages[0].data.toString('utf8'));
    console.log("[SubTest] 取Json資料成功")
  } else {
    let error = "[SubTest] 取Json資料失敗";
    console.log(error)
    return JSON.stringify(ah.ReplyData.NewReplyData(jsonData, error));
  }
  return JSON.stringify(ah.ReplyData.NewReplyData(jsonData, null));

}
