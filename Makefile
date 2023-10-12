# 版本建置Makefile

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
matchmaker: buildMatchmaker deployMatchmaker

# 建構+部屬matchmaker
matchgame: buildMatchgame deployMatchgame
