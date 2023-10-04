@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM Build Image並推上google artifact registry, google放image的地方)

REM 以下為gcr版本，因為gcr逐漸被google淘汰所以就不使用了
REM docker build -t gcr.io/aurafortest/herofishing-matchmaker matchmaker/
REM docker push gcr.io/aurafortest/herofishing-matchmaker
@echo on

docker build -f matchmaker/Dockerfile -t asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchmaker .
docker push asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchmaker
