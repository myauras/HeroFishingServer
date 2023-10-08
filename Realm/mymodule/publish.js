const { PubSub } = require('@google-cloud/pubsub');

// 創建 PubSub 客戶端
const pubsub = new PubSub();
const topicName = 'herofishing-json-topic';


// 通用的發布函數
async function publishJsonData(jsonFileName) {
  const jsonData = require(`./JsonData/${jsonFileName}.json`);
  const dataBuffer = Buffer.from(JSON.stringify(jsonData));

  const messageId = await pubsub.topic(topicName).publishMessage({
    data: dataBuffer,
    attributes: {
      jsonName: jsonFileName  // 使用jsonName欄位指定對應的json資料
    }
  });

  console.log(`已發布${jsonFileName}資料到${topicName}，Message ID：${messageId}`);
}

// 調用函數發布所有JSON資料
async function publishAllData() {
  await publishJsonData('GameSetting');
  await publishJsonData('Hero');
}

publishAllData().catch(console.error);
