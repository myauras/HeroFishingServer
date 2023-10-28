@echo off
Rem 此執行檔是將定義好的Template更新上DB用
Rem 確認已安裝MongoDB Shell, 可以先參考 MongoShell使用說明.txt
Rem 在同目錄中輸入.\Dev_DBTemplate.bat
@echo on
Rem =================Start Updating Tamplete=================
mongosh "mongodb+srv://cluster0.edk0n6b.mongodb.net/?authSource=%%24external&authMechanism=MONGODB-X509" --apiVersion 1 --tls --tlsCertificateKeyFile ".\Keys\MongoDB_X509-cert-1367701216246648381.pem" -f Dev_DBTemplate.js
Rem =================Updating Tamplete Finished=================