# 版本建置Makefile

# 安裝最新的專案Module
BuildModule:
	@echo "==============Start Building Module=============="
	cd mymodule && make all

# 推送App
PushApp:
	@echo "==============Start Pushing App=============="
	cd app && make all

# 完整發布流程
all: BuildModule PushApp