module.exports = {
    NewData: function (data,error) {
        return JSON.stringify({
            Time:new Date(),
            User:context.user,
            Data:data,
            Error:error,
        })
    },
}