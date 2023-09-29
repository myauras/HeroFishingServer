const dbm = require('./DBManager.js');
const gs = require('./GameSetting.js');
module.exports = {
    Log: async function (type, logData, error) {
        let templateData = await GetBaseTemplateData(type, error)
        if (templateData == null) return null;
        ModifyLogData(logData);
        let insertData = Object.assign(templateData, logData);
        await dbm.DB_InsertOne(gs.ColName.gameLog, insertData);
    }
}

async function GetBaseTemplateData(type, error) {
    if (!error) error = null;
    let data = {
        createdAt: new Date(),
        type: type,
        playerID: context.user.id,
        error: error
    }
    return data;
}
function ModifyLogData(logData) {
    // 把_id替換為playerID並移除本來的_id
    if (logData && "_id" in logData) {
        logData["playerID"] = logData._id;
        delete logData["_id"];
    }
}