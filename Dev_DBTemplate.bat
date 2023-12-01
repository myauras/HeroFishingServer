@echo off
Rem 此執行檔是將定義好的Template更新上DB用
Rem 確認已安裝MongoDB Shell, 可以先參考 MongoShell使用說明.txt
Rem 在同目錄中輸入.\Dev_DBTemplate.bat
Rem 官網複製下來的shell連接url中uthSource=%24external&authMechanism=MONGODB-X509 的%要改為%% 否則會出錯
@echo on
Rem =================Start Updating Tamplete=================
mongosh "mongodb+srv://cluster-herofishing.8yp6fou.mongodb.net/?authSource=%%24external&authMechanism=MONGODB-X509" --apiVersion 1 --tls --tlsCertificateKeyFile ".\Keys\aura-developer.pem" -f Dev_DBTemplate.js
Rem =================Updating Tamplete Finished=================