let db = db.getSiblingDB('herofishing');

// let result=db.player.updateOne(
//   { _id: "64eeef1691926d2ac95f7913" },
//   { $set: { point: 188 } }
// )

// let result = db.player.findOneAndUpdate(
//   { _id: "64eeef1691926d2ac95f7913" },
//   { $set: { point: 400 } },
//   { returnNewDocument: true }
// );

let result = db.player.updateOne(
  { _id: "64eeef1691926d2ac95f7913" },
  { $set: { point: 400 } },
  { returnNewDocument: true }
);


printjson(playerDoc);
