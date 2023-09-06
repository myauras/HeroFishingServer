const utility = require('./Utility.js');
module.exports = {
    // 傳入格式
    // data必須為object如果沒有資料要傳就傳{}
    // error為string或null
    NewReplyData: function (data, error) {
        if (!utility.IsNotNullObject(data)) {
            return JSON.stringify({
                Data: null,
                Error: "資料設定錯誤, 回傳的data必須為非null的object",
            });
        }
        return JSON.stringify({
            Data: data,
            Error: error,
        })
    },
}