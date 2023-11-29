# ================================================================
# ===========================Matchmaker===========================
# ================================================================

# Vet專案進行錯誤檢測
vetMatchmaker:
	@echo "==============Vet Matchmaker Module=============="
	go vet matchmaker/...
	go vet herofishingGoModule/...
	@echo "==============Vet Matchmaker Module Finished=============="

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



# 建構+部屬matchmaker
matchmaker: vetMatchmaker autoVersioning-Matchmaker buildMatchmaker deployMatchmaker uploadJsonToServer


# ================================================================
# ===========================Crontasker===========================
# ================================================================

# Vet專案進行錯誤檢測
vetCrontasker:
	@echo "==============Vet Crontasker Module=============="
	go vet crontasker/...
	go vet herofishingGoModule/...
	@echo "==============Vet Crontasker Module Finished=============="

# 自動進版crontasker
autoVersioning-Crontasker:
	@echo "==============AutoVersioning-Crontasker=============="
	powershell -ExecutionPolicy Bypass -File .\Dev_AutoVersioning-Crontasker.ps1
	@echo "==============AutoVersioning-Crontasker Finished=============="

# 建構crontasker
buildCrontasker:
	@echo "==============Start Building Crontasker=============="
	.\Dev_BuildCrontasker.bat
	@echo "==============Crontasker Build Finished=============="

# 部屬crontasker
deployCrontasker:
	@echo "==============Start Deploy Crontasker=============="
	.\Dev_DeployCrontasker.bat
	@echo "==============Crontasker Deploy Finished=============="




# 建構+部屬crontasker
crontasker: vetCrontasker autoVersioning-Crontasker buildCrontasker deployCrontasker



# ================================================================
# ===========================Matchgame============================
# ================================================================

# Vet專案進行錯誤檢測
vetMatchgame:
	@echo "==============Vet Matchgame Module=============="
	go vet matchgame/...
	go vet herofishingGoModule/...
	@echo "==============Vet Matchgame Module Finished=============="

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

# 移除matchgame舊版本pods
deleteMatchgameOldPods:
	@echo "==============Start Delete Old Matchgame Pods=============="
	powershell -ExecutionPolicy Bypass -File .\Dev_DeleteAllMatchgameAndKeepByVersion.ps1
	@echo "==============Matchgame Delete Finished=============="

# 更新Json到GCS上
uploadJsonToServer:
	@echo "==============Uploading Json Datas to GCS=============="
	.\Dev_UploadJsonToServer.bat
	@echo "==============Upload Finished=============="


# 建構+部屬matchgame
matchgame: vetMatchgame autoVersioning-Matchgame buildMatchgame deployMatchgame deleteMatchgameOldPods uploadJsonToServer
