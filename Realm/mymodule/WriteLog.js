const dbm = require('./DBManager.js');
const gs = require('./GameSetting.js');
module.exports = {
    Log: async function (type, logData, error) {
        let myData = await GetBaseTemplateData(type, error)
        if (myData == null) return null;
        let insertData = Object.assign(myData, logData);
        await dbm.DB_InsertOne(gs.ColName.gameLog, insertData);
        return doc;
    }
}

async function GetBaseTemplateData(type, error) {
    if (!error) error = null;
    let data = {
        createAt: new Date(),
        type: type,
        playerID: context.user.id,
        error: error
    }
    return data;
}