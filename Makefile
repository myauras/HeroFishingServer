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


# 建構+部屬matchmaker
matchmaker: autoVersioning-Matchmaker buildMatchmaker deployMatchmaker


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
deleveMatchgameOldPods:
	@echo "==============Start Delete Old Matchgame Pods=============="
	powershell -ExecutionPolicy Bypass -File .\Dev_DeleteAllMatchgameAndKeepByVersion.ps1
	@echo "==============Matchgame Delete Finished=============="


# 建構+部屬matchgame
matchgame: autoVersioning-Matchgame buildMatchgame deployMatchgame
