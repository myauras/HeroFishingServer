# 版本建置Makefile

# 自動進版matchmaker
autoVersioning-Matchmaker:
	@echo "==============AutoVersioning-Matchmaker=============="
	.\Dev_AutoVersioning-Matchmaker.bat
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


# 自動進版matchmaker
autoVersioning-Matchgame:
	@echo "==============AutoVersioning-Matchgame=============="
	.\Dev_AutoVersioning-Matchgame.bat
	@echo "==============AutoVersioning-Matchgame Finished=============="

# 建構matchgame
buildMatchgame:
	@echo "==============Start Building Matchmaker=============="
	.\Dev_BuildMatchgame.bat
	@echo "==============Matchmaker Build Finished=============="

# 部屬matchgame
deployMatchgame:
	@echo "==============Start Deploy Matchmaker=============="
	.\Dev_DeployMatchgame.bat
	@echo "==============Matchmaker Deploy Finished=============="

# 建構+部屬matchmaker
matchmaker: autoVersioning-Matchmaker buildMatchmaker deployMatchmaker

# 建構+部屬matchmaker
matchgame: autoVersioning-Matchgame buildMatchgame deployMatchgame
