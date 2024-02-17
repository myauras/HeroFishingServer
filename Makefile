# ================================================================
# ===========================Common===============================
# ================================================================

# 更新Json到GCS上(只要Json有更新都要重新佈署)
gamejson:
	@echo "==============Uploading Json Datas to GCS=============="
	.\Dev_UploadJsonToServer.bat
	@echo "==============Upload Finished=============="

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
	powershell -ExecutionPolicy Bypass -File .\CICD_Matchmaker_Dev\Dev_AutoVersioning-Matchmaker.ps1
	@echo "==============AutoVersioning-Matchmaker Finished=============="

# 建構matchmaker
buildMatchmaker:
	@echo "==============Start Building Matchmaker=============="
	.\CICD_Matchmaker_Dev\Dev_BuildMatchmaker.bat
	@echo "==============Matchmaker Build Finished=============="

# 部屬matchmaker
deployMatchmaker:
	@echo "==============Start Deploy Matchmaker=============="
	.\CICD_Matchmaker_Dev\Dev_DeployMatchmaker.bat
	@echo "==============Matchmaker Deploy Finished=============="



# 建構+部屬matchmaker
matchmaker: vetMatchmaker autoVersioning-Matchmaker buildMatchmaker deployMatchmaker


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
	powershell -ExecutionPolicy Bypass -File .\CICD_Crontasker_Dev\Dev_AutoVersioning-Crontasker.ps1
	@echo "==============AutoVersioning-Crontasker Finished=============="

# 建構crontasker
buildCrontasker:
	@echo "==============Start Building Crontasker=============="
	.\CICD_Crontasker_Dev\Dev_BuildCrontasker.bat
	@echo "==============Crontasker Build Finished=============="

# 部屬crontasker
deployCrontasker:
	@echo "==============Start Deploy Crontasker=============="
	.\CICD_Crontasker_Dev\Dev_DeployCrontasker.bat
	@echo "==============Crontasker Deploy Finished=============="

# 移除crontasker舊版本pods
deleteCrontaskerOldPods:
	@echo "==============Start Delete Old Crontasker Pods=============="
	powershell -ExecutionPolicy Bypass -File .\CICD_Crontasker_Dev\Dev_DeleteAllCrontaskerAndKeepByVersion.ps1
	@echo "==============Crontasker Delete Finished=============="




# 建構+部屬crontasker
crontasker: vetCrontasker autoVersioning-Crontasker buildCrontasker deployCrontasker deleteCrontaskerOldPods



# ================================================================
# ===========================Lobby================================
# ================================================================

# Vet專案進行錯誤檢測
vetLobby:
	@echo "==============Vet Lobby Module=============="
	go vet lobby/...
	go vet herofishingGoModule/...
	@echo "==============Vet Lobby Module Finished=============="

# 自動進版lobby
autoVersioning-Lobby:
	@echo "==============AutoVersioning-Lobby=============="
	powershell -ExecutionPolicy Bypass -File .\CICD_Lobby_Dev\Dev_AutoVersioning-Lobby.ps1
	@echo "==============AutoVersioning-Lobby Finished=============="

# 建構lobby
buildLobby:
	@echo "==============Start Building Lobby=============="
	.\CICD_Lobby_Dev\Dev_BuildLobby.bat
	@echo "==============Lobby Build Finished=============="

# 部屬lobby
deployLobby:
	@echo "==============Start Deploy Lobby=============="
	.\CICD_Lobby_Dev\Dev_DeployLobby.bat
	@echo "==============Lobby Deploy Finished=============="


# 建構+部屬lobby
lobby: vetLobby autoVersioning-Lobby buildLobby deployLobby


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
	powershell -ExecutionPolicy Bypass -File .\CICD_Matchgame_Dev\Dev_AutoVersioning-Matchgame.ps1
	@echo "==============AutoVersioning-Matchgame Finished=============="

# 建構matchgame
buildMatchgame:
	@echo "==============Start Building Matchgame=============="
	.\CICD_Matchgame_Dev\Dev_BuildMatchgame.bat
	@echo "==============Matchgame Build Finished=============="

# 部屬matchgame
deployMatchgame:
	@echo "==============Start Deploy Matchgame=============="
	.\CICD_Matchgame_Dev\Dev_DeployMatchgame.bat
	@echo "==============Matchgame Deploy Finished=============="

# 部屬matchgame-testver
deployMatchgame-testver:
	@echo "==============Start Deploy Matchgame=============="
	.\CICD_Matchgame_Dev\Dev_DeployMatchgameTestVer.bat
	@echo "==============Matchgame Deploy Finished=============="

# 移除matchgame舊版本pods
deleteMatchgameOldPods:
	@echo "==============Start Delete Old Matchgame Pods=============="
	powershell -ExecutionPolicy Bypass -File .\CICD_Matchgame_Dev\Dev_DeleteAllMatchgameAndKeepByVersion.ps1
	@echo "==============Matchgame Delete Finished=============="


# 建構+部屬matchgame
matchgame: vetMatchgame autoVersioning-Matchgame buildMatchgame deployMatchgame deleteMatchgameOldPods
# 建構+部屬matchgame-testver
matchgame-testver: vetMatchgame autoVersioning-Matchgame buildMatchgame deployMatchgame-testver
