const gs = require('./GameSetting.js');
module.exports = {
    // API與回傳格式可參考官方文件: https://www.mongodb.com/docs/v5.2/crud/
    // 單筆插入
    DB_InsertOne: async function (colName,data) {
        const col=GetCol(colName);     
        data= GetTemplateData(col,colName)
        if(data==null)
            return null;
        return await col.insertOne(data);
    },
}
function GetCol(colName){
    if(!(colName in gs.ColName)){
        console.log("GetCol傳入尚未定義的集合: ${colName}");
        return null;
    }
    const cluster = context.services.get("Cluster0");
    if(!cluster){
        console.log("無此cluster");
        return null;
    }

    const db = cluster.db("herofishing");
    if(!db){
        console.log("無此db");
        return null;
    }
    const col = db.collection(colName);
    if(!col){
        console.log("無此collection: ${colName}");
        return null;
    }
    return col;
}
// 依據模板初始化文件欄位, 在GameSetting中的ColTemplate若有定義傳入的集合就會使用DB上的模板資料
// 模板資料可以透過 環境版本_DBTemplate.bat 那份檔案來部署到DB上
async function GetTemplateData(col,colName){
    if(!col){
        console.log("GetTemplateData傳入null col");
        return null;
    }
    if(!(colName in gs.ColName)){
        console.log("GetTemplateData傳入尚未定義的集合: ${colName}");
        return null;
    }

    // 取得doc基本資料
    let data= GetBaseTemplateData();

    // 若沒有定義模板就直接回傳data
    if(!(colName in gs.ColTemplate))
        return data;

    // 取得DB上的模板
    const doc = await col.findOne({ "_id": "player" });
    
    return data;
}

// 取得doc基本資料
function GetBaseTemplateData(){
    let data={
        "createdAt":new Date(),
    }
    return data;
}
// playerDoc加入必要欄位
function SetPlayerDocBaseData(data){
    let nowDate=new Date();
    // 註冊方式
    if(!("authType" in data)) 
        data["authType"]=gs.AuthType.Unknown;
    // 點數
    if(!("point" in data)) 
        data["point"]=Int64(0);
    // 上線狀態
    if(!("onlineState" in data)) 
        data["onlineState"]=gs.OnlineState.Offline;
    // 上次登入時間
    if(!("lastSignin" in data)) 
        data["lastSignin"]=nowDate;
    // 上次登出時間
    if(!("lastSignout" in data)) 
        data["lastSignout"]=nowDate;
    // 是否有被封鎖
    if(!("ban" in data)) 
        data["ban"]=false;
    // 登入裝置ID
    if(!("deviceUID" in data)) 
        data["deviceUID"]="";
    return data;
}