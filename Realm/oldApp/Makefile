# 版本建置Makefile

# 安裝最新的專案Module
InstallModule:
	@echo "==============Start Installing Module=============="
	cd functions && npm install aura-herofishing@latest

# 推送App
PushApp:
	@echo "==============Start Pushing App=============="
	realm-cli push --include-package-json

# 重新連結aura-herofishin
RelinkModule:
	@echo "=============Relink npm module=============="
	cd functions && npm link aura-herofishing
# 完整發布流程
all: InstallModule PushApp RelinkModule