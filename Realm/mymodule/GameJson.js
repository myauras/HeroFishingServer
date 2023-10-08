// const { PubSub } = require('@google-cloud/pubsub');

module.exports = {
    Get: async function (jsonName) {
        try {
            const jsonData = require(`./JsonData/${jsonName}.json`);
            return jsonData;
        } catch (error) {
            throw new Error(`取得 ${jsonName} 時發生錯誤: ${error.message}`);
        }
    }
    // Get: async function (jsonName) {
    //     console.log("取" + jsonName + "資料");
    //     const projectId = 'aurafortest';
    //     const topicName = 'herofishing-json-topic';
    //     const subscriptionName = 'herofishing-subscription';

    //     const pubsub = new PubSub({ projectId });
    //     const topic = pubsub.topic(topicName);
    //     const subscription = topic.subscription(subscriptionName);
    //     const timeOutMS = 10000; // 10秒超時

    //     // 檢查是否已經訂閱了
    //     const [subscriptions] = await pubsub.getSubscriptions();
    //     console.log("a");
    //     const exists = subscriptions.some(sub => sub.name.endsWith(subscriptionName));
    //     if (!exists) {
    //         try {
    //             await topic.createSubscription(subscriptionName);
    //         } catch (error) {
    //             if (error.code !== 6) {  // 已訂閱
    //                 throw new Error(`訂閱失敗: ${error}`);
    //             }
    //         }
    //     }
    //     console.log("b");

    //     // 取訂閱內容
    //     try {
    //         console.log("c");
    //         const [message] = await Promise.race([
    //             new Promise(resolve => subscription.once('message', resolve)),
    //             new Promise((_, reject) => setTimeout(() => reject(new Error("GameJson取資料超時")), timeOutMS))
    //         ]);
    //         console.log("d");
    //         if (message.attributes.jsonName === jsonName) {
    //             message.ack();
    //             return JSON.parse(message.data.toString());
    //         } else {
    //             throw new Error(`錯誤的jsonName 可能是忘記publish了 : ${message.attributes.jsonName}`);
    //         }
    //         console.log("e");
    //     } catch (error) {
    //         throw new Error(`接收訊息失敗: ${error.message}`);
    //     }
    //     console.log("f");
    // }
};
