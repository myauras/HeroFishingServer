const { PubSub } = require('@google-cloud/pubsub');

async function ensureTopicAndSubscription() {
  const pubsub = new PubSub();
  const topicName = 'herofishing-json-topic';
  const subscriptionName = 'herofishing-json-subscription';

  // 檢查或創建主題
  let [topics] = await pubsub.getTopics();
  let topicExists = topics.some(topic => topic.name.endsWith(`/${topicName}`));
  let topic;
  if (topicExists) {
    topic = pubsub.topic(topicName);
  } else {
    [topic] = await pubsub.createTopic(topicName);
  }

  // 檢查或創建訂閱
  let [subscriptions] = await pubsub.getSubscriptions();
  let subscriptionExists = subscriptions.some(subscription => subscription.name.endsWith(`/${subscriptionName}`));
  if (!subscriptionExists) {
    await topic.createSubscription(subscriptionName);
  }
}

// 通用的發布函數
async function publishJsonData(jsonFileName) {
  const pubsub = new PubSub();
  const topicName = 'herofishing-json-topic';
  const jsonData = require(`./JsonData/${jsonFileName}.json`);
  const dataBuffer = Buffer.from(JSON.stringify(jsonData));

  const messageId = await pubsub.topic(topicName).publishMessage({
    data: dataBuffer,
    attributes: {
      jsonName: jsonFileName
    }
  });

  console.log(`已發布${jsonFileName}資料到${topicName}，Message ID：${messageId}`);
}

// 呼叫函數發布所有JSON資料
async function publishAllData() {
  await ensureTopicAndSubscription();  // 確保主題和訂閱存在
  await publishJsonData('GameSetting');
  await publishJsonData('Hero');
}

publishAllData().catch(console.error);
