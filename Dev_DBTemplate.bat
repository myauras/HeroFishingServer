Rem 此執行檔是將定義好的Template更新上DB用

Rem 確認已安裝MongoDB Shell, 可以先參考 MongoRealm專案說明.txt

Rem 寫入Tamplete
mongosh "mongodb+srv://cluster0.edk0n6b.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509" --apiVersion 1 --tls --tlsCertificateKeyFile ".\Keys\X509-cert-6644120643259731564.pem" -f Dev_DBTemplate.js

