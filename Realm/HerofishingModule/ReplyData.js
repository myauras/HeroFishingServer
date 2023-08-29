const utility = require('./Utility.js');
module.exports = {
    NewReplyData: function (data, error) {
        if (!utility.IsObject(data)) {
            return JSON.stringify({
                Data: null,
                Error: "資料設定錯誤, 回傳的data必須為object",
            });
        }
        return JSON.stringify({
            Data: data,
            Error: error,
        })
    },
}