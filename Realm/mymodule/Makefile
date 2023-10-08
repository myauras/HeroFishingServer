# 版本建置Makefile

# 增加package.json版號並存檔
increment_version:
	@echo "==============Incrementing version=============="
	@node -p -e "const fs = require('fs'); \
	             let package = JSON.parse(fs.readFileSync('./package.json')); \
	             let version = package.version.split('.'); \
	             version[2] = Number(version[2]) + 1; \
	             package.version = version.join('.'); \
	             fs.writeFileSync('./package.json', JSON.stringify(package, null, 2)); \
	             package.version"

# 發布到npm上
publish_npm:
	@echo "==============Publishing to npm=============="
	npm publish

# 執行google publish
publish_googlepub:
	@echo "==============google publish=============="
	node publish.js

	

# 完整發布流程
all: increment_version publish_npm publish_googlepub
