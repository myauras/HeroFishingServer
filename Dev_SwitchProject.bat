@echo off
REM 使用此批次檔切換到Dev環境 在powershell中執行.\Dev_SwitchProject.bat
@echo on
gcloud config set project testgcpproject1-415003
gcloud config set container/cluster cluster-herofishing
gcloud container clusters get-credentials cluster-herofishing --zone=asia-east1-c
kubectl config current-context



