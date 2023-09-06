const gs = require('./GameSetting.js');
module.exports = {
    // API可參考官方文件: https://www.mongodb.com/docs/atlas/app-services/functions/mongodb/api/
    // Query Selector可參考: https://www.mongodb.com/docs/manual/reference/operator/query/#query-selectors
    // Update Operators可參考: https://www.mongodb.com/docs/manual/reference/operator/update/

    // 單筆插入
    // 返回格式為:
    // 1. 錯誤就會返回null
    // 2. 成功就會返回doc
    DB_InsertOne: async function (colName, insert) {
        if (!colName || !insert) {
            console.log(`[DBManager] 傳入資料錯誤 colName=${colName} , insert=${insert}`);
            return null;
        }
        let templateData = await GetTemplateData(colName)
        if (templateData == null) return null;
        let insertData = Object.assign(templateData, insert);
        col = GetCol(colName);
        if (!col) return null;
        let result = await col.insertOne(insertData);
        let doc = GetInsertResult(insertData, result);
        return doc;
    },
    // 單筆查找
    // projection是返回doc需要的欄位，傳入null就是回傳找到的整份doc
    // 返回格式為:
    // 1. 錯誤就會返回null
    // 2. 成功就會返回doc
    DB_FindOne: async function (colName, query, projection) {
        if (!colName || !query) {
            console.log(`[DBManager] 傳入資料錯誤 colName=${colName} , query=${query}`);
            return null;
        }
        col = GetCol(colName);
        if (!col) return null;
        let doc = await col.findOne(query, projection);
        // findOne的回傳格式直接是文件, 可以直接回傳
        return doc;
    },
    // 單筆更新
    // 返回格式為:
    // 1. 錯誤就會返回false
    // 2. 成功就會返回true
    DB_UpdateOne: async function (colName, query, update, options) {
        if (!colName || !query || !update) {
            console.log(`[DBManager] 傳入資料錯誤 colName=${colName} , query=${query} , update=${update}`);
            return null;
        }
        col = GetCol(colName);
        if (!col) return null;
        if (!options) {
            options = { upsert: false };
        }
        let result = await col.updateOne(query, update, options);
        let success = GetUpdateOneResult(myData, result);
        return success;
    },
    // 單筆查找並更新
    // 返回格式為:
    // 1. 錯誤就會返回null
    // 2. 插入成功就會返回更新後的doc
    DB_FindeOneAndUpdate: async function (colName, query, update, options) {
        if (!colName || !query || !update) {
            console.log(`[DBManager] 傳入資料錯誤 colName=${colName} , query=${query} , update=${update}`);
            return null;
        }
        col = GetCol(colName);
        if (!col) return null;
        if (!options) {
            options = {
                returnNewDocument: true,
                upsert: false
            };
        }
        let doc = await col.findOneAndUpdate(query, update, options);
        // findOneAndUpdate的回傳格式直接是文件, 可以直接回傳
        return doc;
    },
}
function GetAtlas() {
    return context.services.get("mongodb-atlas");
}
function GetDB() {
    const atlas = GetAtlas();
    if (!atlas) {
        console.log("[DBManager] 無此atlas");
        return null;
    }
    const db = atlas.db("herofishing")//db不存在也會拋出一個可用的db，不會拋出錯誤或null
    return db;
}
function GetCol(colName) {
    if (!(colName in gs.ColName)) {
        console.log(`[DBManager] GetCol傳入尚未定義的集合: ${colName}`);
        return null;
    }
    const db = GetDB();
    const col = db.collection(colName);//collection不存在也會拋出一個可用的collection，不會拋出錯誤或null
    return col;
}
function GetInsertResult(doc, result) {
    // result格式是這樣
    // {   
    //     "insertedId" : ObjectId("5fb3e0ee04f507136c837a7b")
    //   }    
    if (!result) return null;
    if (!("insertedId" in result)) return null;
    doc._id = result["insertedId"];
    return doc;
}
function GetUpdateOneResult(result) {
    // result格式是這樣
    // {
    //     acknowledged: true,
    //     insertedId: null,
    //     matchedCount: 1,
    //     modifiedCount: 0,
    //     upsertedCount: 0
    // }
    if (!result) return false;
    return result["acknowledged"];
}

// 依據模板初始化文件欄位, 在GameSetting中的ColTemplate若有定義傳入的集合就會使用DB上的模板資料
// 模板資料可以透過 環境版本_DBTemplate.bat 那份檔案來部署到DB上
async function GetTemplateData(colName) {
    if (!(colName in gs.ColName)) {
        console.log(`[DBManager] GetTemplateData傳入尚未定義的集合: ${colName}`);
        return null;
    }

    // 取得doc基本資料
    let data = GetBaseTemplateData();
    // 若沒有定義模板就直接回傳data
    if (!gs.ColTemplate.has(colName)) return data;

    // 取得DB上的模板並使用模板資料
    const templateCol = GetCol(gs.ColName.template);
    if (!templateCol) return data;
    const templateDoc = await templateCol.findOne({ "_id": colName });

    if (!templateDoc) {// 找不到模板就直接返回目前的data
        console.log(`[DBManager] 有定義模板, 但找不到模板資料: ${colName}`);
        return data;
    }
    // 刪除不需要使用的模板資料
    const keysToDelete = ['_id', 'createdAt'];
    for (let key of keysToDelete) {
        delete templateDoc[key];
    }
    // 使用模板資料
    let nowDate = new Date();
    for (let key in templateDoc) {
        if (!(key in data)) {
            let suffix = "_nowDate";// 帶有_nowDate結尾的欄位要改成抓現在時間
            if (key.endsWith(suffix)) {
                let newKey = key.slice(0, -suffix.length);
                data[newKey] = nowDate;
            } else
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