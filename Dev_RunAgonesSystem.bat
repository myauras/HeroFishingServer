@echo off
REM 使用此批次檔執行Agones服務 在powershell中執行.\Dev_RunAgonesSystem.bat
REM 詳細的Agones服務建立可以參考官方文件 https://agones.dev/site/docs/installation/install-agones/helm/
@echo on
helm repo add agones https://agones.dev/chart/stable
helm repo update
helm install agones-release --namespace agones-system --create-namespace agones/agones