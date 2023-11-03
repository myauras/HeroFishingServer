# 版本建置Makefile

# 自動進版matchmaker
autoVersioning-Matchmaker:
	@echo "==============AutoVersioning-Matchmaker=============="
	powershell -ExecutionPolicy Bypass -File .\Dev_AutoVersioning-Matchmaker.ps1
	@echo "==============AutoVersioning-Matchmaker Finished=============="

# 建構matchmaker
buildMatchmaker:
	@echo "==============Start Building Matchmaker=============="
	.\Dev_BuildMatchmaker.bat
	@echo "==============Matchmaker Build Finished=============="

# 部屬matchmaker
deployMatchmaker:
	@echo "==============Start Deploy Matchmaker=============="
	.\Dev_DeployMatchmaker.bat
	@echo "==============Matchmaker Deploy Finished=============="

# Vet專案進行錯誤檢測
vetMatchmaker:
	@echo "==============Vet Matchmaker Module=============="
	go vet matchmaker/...
	go vet herofishingGoModule/...
	@echo "==============Vet Matchmaker Module Finished=============="


# 建構+部屬matchmaker
matchmaker: vetMatchmaker autoVersioning-Matchmaker buildMatchmaker deployMatchmaker uploadJsonToServer


# 自動進版matchgame
autoVersioning-Matchgame:
	@echo "==============AutoVersioning-Matchgame=============="	
	powershell -ExecutionPolicy Bypass -File .\Dev_AutoVersioning-Matchgame.ps1
	@echo "==============AutoVersioning-Matchgame Finished=============="

# 建構matchgame
buildMatchgame:
	@echo "==============Start Building Matchgame=============="
	.\Dev_BuildMatchgame.bat
	@echo "==============Matchgame Build Finished=============="

# 部屬matchgame
deployMatchgame:
	@echo "==============Start Deploy Matchgame=============="
	.\Dev_DeployMatchgame.bat
	@echo "==============Matchgame Deploy Finished=============="

# Vet專案進行錯誤檢測
vetMatchgame:
	@echo "==============Vet Matchgame Module=============="
	go vet matchgame/...
	go vet herofishingGoModule/...
	@echo "==============Vet Matchgame Module Finished=============="

# 移除matchgame舊版本pods
deleteMatchgameOldPods:
	@echo "==============Start Delete Old Matchgame Pods=============="
	powershell -ExecutionPolicy Bypass -File .\Dev_DeleteAllMatchgameAndKeepByVersion.ps1
	@echo "==============Matchgame Delete Finished=============="

uploadJsonToServer:
	@echo "==============Uploading Json Datas to GCS=============="
	.\Dev_UploadJsonToServer.bat
	@echo "==============Upload Finished=============="



# 建構+部屬matchgame
matchgame: vetMatchgame autoVersioning-Matchgame buildMatchgame deployMatchgame deleteMatchgameOldPods uploadJsonToServer
