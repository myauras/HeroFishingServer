module.exports = {
    // 確認是否為object(null也算object)
    IsObject: function (obj) {
        return typeof obj === 'object' && obj.constructor === Object;
    },
    // 確認是否為非null object
    IsNotNullObject: function (obj) {
        return typeof obj === 'object' && obj !== null && obj.constructor === Object;
    },
}