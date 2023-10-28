const { MongoClient } = require('mongodb');

async function dbMigration() {
  // 來源專案
  const uriA = 'mongodb+srv://tester:test123@cluster0.edk0n6b.mongodb.net/?retryWrites=true&w=majority';
  // 目標專案
  const uriB = 'mongodb+srv://aura:dw0ivy2ljTXuoZGW@cluster-herofishing.8yp6fou.mongodb.net/?retryWrites=true&w=majority';

  // 連線來源專案
  const clientA = new MongoClient(uriA);
  await clientA.connect();

  // 連線目標專案
  const clientB = new MongoClient(uriB);
  await clientB.connect();

  // 取得來源專案的DB
  const dbA = clientA.db('herofishing');

  // 取得目標專案的DB
  const dbB = clientB.db('herofishing');

  // 取得來源專案的所有Collections
  const collections = await dbA.listCollections().toArray();

  // 遍歷來源專案的Collections
  for (const collectionInfo of collections) {
    const collectionName = collectionInfo.name;
    const collectionA = dbA.collection(collectionName);
    const collectionB = dbB.collection(collectionName);
    const cursor = collectionA.find({});
    // 將docs全部複製到目標專案
    while (await cursor.hasNext()) {
      const document = await cursor.next();
      await collectionB.insertOne(document);
    }
    console.log("完成複製Collection: " + collectionName);
  }

  await clientA.close();
  await clientB.close();
}

// 例外擷取
dbMigration().catch(console.error);
