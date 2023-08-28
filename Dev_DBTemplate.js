var db = db.getSiblingDB('herofishing');
db.template.insertOne(
  {
    _id: "player",
    createdAt: new Date(),
    authType: "Guest",
    point: NumberLong(1),
    onlineState: "Offline",
    lastSignin_nowDate: null,
    lastSignout_nowDate: null,
    ban: false,
    deviceUID: "",
  }
);
