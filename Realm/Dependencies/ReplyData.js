module.exports = {
    NewData: function (data,error) {
        console.log("aaaaaaaaa")
        return JSON.stringify({
            Time:new Date(),
            User:context.user,
            Data:data,
            Error:error,
        })
    },
}