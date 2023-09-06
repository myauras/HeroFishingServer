const dbm = require('./DBManager.js');
const gs = require('./GameSetting.js');
module.exports = {
    Log: async function (type, logData, error) {
        let templateData = await GetBaseTemplateData(type, error)
        if (templateData == null) return null;
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