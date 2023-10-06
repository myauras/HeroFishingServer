const GameSettingJson = require('./GameSetting.json');

const {PubSub} = require('@google-cloud/pubsub');

// 創建 PubSub 客戶端
const pubsub = new PubSub();

// 主題名稱
const topicName = 'herofishing-json-topic';

// 將 JSON 數據發布到主題
async function publishJsonData() {
  const dataBuffer = Buffer.from(JSON.stringify(GameSettingJson));
  
  const messageId = await pubsub.topic(topicName).publish(dataBuffer);
  console.log(`JSON資料已發布，Message ID：${messageId}`);
}

publishJsonData();