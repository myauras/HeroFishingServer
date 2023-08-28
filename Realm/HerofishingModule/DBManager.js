const gm = require('./GameSetting.js');
module.exports = {
    // API與回傳格式可參考官方文件: https://www.mongodb.com/docs/v5.2/reference/method/db.collection.insertOne/
    DB_GetCol: function(colName){
        const cluster = context.services.get("Cluster0");
        const db = cluster.db("herofishing");
        const playerCol = db.collection(colName);
        return playerCol;
    },
    // 單筆插入
    DB_InsertOne: async function (colName,data) {
        const col=this.GetCol(colName);
        data= GetInsertData(colName,data)
        if(data==null)
            return null;
        return await col.insertOne(data);
    },
}
function GetInsertData(colName, data){
    switch(colName){
        case gm.ColName.Player:
            AddPlayerDocBaseData(data)
            break;
        default:
            console.log("GetInsertData的colName尚未定義: ${colName}");
            return null;
    }
    return data;
}

// doc加入必要欄位
function AddDocBaseData(data){
    data["createdAt"]=new Date();
    return data;
}
// playerDoc加入必要欄位
function AddPlayerDocBaseData(data){
    let nowDate=new Date();
    AddDocBaseData(data)
    // 註冊方式
    if(!("authType" in data)) 
        data["authType"]=gm.AuthType.Unknown;
    // 點數
    if(!("point" in data)) 
        data["point"]=Int64(0);
    // 上線狀態
    if(!("onlineState" in data)) 
        data["onlineState"]=gm.OnlineState.Offline;
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