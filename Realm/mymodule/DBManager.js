const gs = require('./GameSetting.js');
module.exports = {
    // API與回傳格式可參考官方文件: https://www.mongodb.com/docs/v5.2/crud/

    // 單筆插入
    // 返回格式為:
    // 1. 錯誤就會返回null
    // 2. 插入成功就會返回{id:_id,data:doc}
    DB_InsertOne: async function (colName, data) {
        let myData = await GetTemplateData(colName)
        if (myData == null) return null;
        Object.assign(myData, data);
        col = GetCol(colName);
        if (!col) return null;
        let result = await col.insertOne(myData);
        let resultData = GetInsertResult(myData, result);
        return resultData;
    },
}
function GetCluster() {
    return context.services.get("mongodb-atlas");
}
function GetDB() {
    const cluster = GetCluster();
    if (!cluster) {
        console.log("無此cluster");
        return null;
    }
    const db = cluster.db("herofishing")
    if (!db) {
        console.log("無此db");
        return null;
    }
    return db;
}
function GetCol(colName) {
    if (!(colName in gs.ColName)) {
        console.log(`GetCol傳入尚未定義的集合: ${colName}`);
        return null;
    }
    const db = GetDB();
    if (!db) {
        console.log("無此db");
        return null;
    }
    const col = db.collection(colName);
    if (!col) {
        console.log(`無此collection: ${colName}`);
        return null;
    }
    return col;
}
function GetInsertResult(data, result) {
    // result格式是這樣
    // {
    //     "acknowledged" : true,
    //     "insertedId" : ObjectId("5fb3e0ee04f507136c837a7b")
    //   }
    if (!result) return null;
    if (result["acknowledged"] == false) return null;
    let newData = {
        id: result["insertedId"],
        data: data,
    }
    return newData;
}

// 依據模板初始化文件欄位, 在GameSetting中的ColTemplate若有定義傳入的集合就會使用DB上的模板資料
// 模板資料可以透過 環境版本_DBTemplate.bat 那份檔案來部署到DB上
async function GetTemplateData(colName) {
    if (!(colName in gs.ColName)) {
        console.log(`GetTemplateData傳入尚未定義的集合: ${colName}`);
        return null;
    }

    // 取得doc基本資料
    let data = GetBaseTemplateData();

    // 若沒有定義模板就直接回傳data
    if (!(colName in gs.ColTemplate)) return data;

    // 取得DB上的模板並使用模板資料
    const templateCol = GetCol(gs.ColName.template);
    if (!templateCol) return data;
    const templateDoc = await templateCol.findOne({ "_id": colName });
    if (!templateDoc) {// 找不到模板就直接返回目前的data
        console.log(`有定義模板, 但找不到模板資料: ${colName}`);
        return data;
    }
    // 刪除不需要使用的模板資料
    const keysToDelete = ['_id', 'createAt'];
    for (let key of keysToDelete) {
        delete templateDoc[key];
    }
    // 使用模板資料
    let nowDate = new Date();
    for (let key in templateDoc) {
        if (!(key in data)) {
            if (templateDoc[key].endsWith("_nowDate")) templateDoc[key] = nowDate;
            data[key] = templateDoc[key];
        }
    }
    return data;
}

// 取得doc基本資料
function GetBaseTemplateData() {
    let data = {
        "createdAt": new Date(),
    }
    return data;
}