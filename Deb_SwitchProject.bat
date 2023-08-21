@echo off
REM 使用此批次檔切換到Dev環境 在powershell中執行.\Deb_SwitchProject.bat
@echo on
call gcloud config set project aurafortest
call gcloud config set container/cluster gameserver-cluster
call gcloud container clusters get-credentials gameserver-cluster --zone=asia-east1-a
call kubectl config current-context