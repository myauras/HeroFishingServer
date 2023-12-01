const redis = require('redis');
const client = redis.createClient();

module.exports = {
    InsertOne,
    FindOne,
    UpdateOne
};


client.on('error', function (err) {
    console.log('Redis Client Error', err);
});

client.connect();

// 插入資料
async function InsertOne(key, value) {
    try {
        await client.set(key, JSON.stringify(value));
        return value;
    } catch (error) {
        console.log(`[DBManager] 插入數據錯誤: ${error}`);
        return null;
    }
}

// 查找資料
async function FindOne(key) {
    try {
        const value = await client.get(key);
        return value ? JSON.parse(value) : null;
    } catch (error) {
        console.log(`[DBManager] 查找數據錯誤: ${error}`);
        return null;
    }
}

// 更新資料
async function UpdateOne(key, update) {
    try {
        const existingValue = await client.get(key);
        if (!existingValue) {
            return false;
        }
        const newValue = { ...JSON.parse(existingValue), ...update };
        await client.set(key, JSON.stringify(newValue));
        return true;
    } catch (error) {
        console.log(`[DBManager] 更新數據錯誤: ${error}`);
        return false;
    }
}

