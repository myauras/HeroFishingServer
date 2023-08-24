exports = async function Signup() {
    data={
        user:context.user.id,
        msg:"signup from server",
    }
    return JSON.stringify(data)
  }