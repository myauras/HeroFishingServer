exports = async function InitPlayerData() {
    // if(context.user.id==""){
    //     console.log("context.user.id is empty")
    //     console.log(JSON.stringify(context.user))
    //     return 
    // }
    const manager = require("aurafortest-herofishing")
    console.log(JSON.stringify(manager.NewData("A","b")))
    return JSON.stringify(manager.NewData("A","b"))
  }
