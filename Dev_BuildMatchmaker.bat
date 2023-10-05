@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM Build Image並推上google artifact registry, google放image的地方)

REM 以下為gcr版本，因為gcr逐漸被google淘汰所以就不使用了
REM docker build -t gcr.io/aurafortest/herofishing-matchmaker matchmaker/
REM docker push gcr.io/aurafortest/herofishing-matchmaker


REM 如果puch image發生錯誤可以跑以下重新登入跟認證流程試試看
@REM gcloud auth revoke
@REM gcloud auth login
@REM docker logout asia-east1-docker.pkg.dev
@REM gcloud auth configure-docker asia-east1-docker.pkg.dev

@echo on


REM =======change go.mod for docker setting=======
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchmaker\go.mod) | ForEach-Object { $_ -replace 'replace herofishingGoModule => ../herofishingGoModule // for local', '// replace herofishingGoModule => ../herofishingGoModule // for local' } | Set-Content matchmaker\go.mod"
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchmaker\go.mod) | ForEach-Object { $_ -replace '// replace herofishingGoModule => /go/src/herofishingGoModule // for docker', 'replace herofishingGoModule => /go/src/herofishingGoModule // for docker' } | Set-Content matchmaker\go.mod"

REM =======build image=======
docker build -f matchmaker/Dockerfile -t asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchmaker .
REM =======push image=======
docker push asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchmaker

REM =======change go.mod back to local setting=======
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchmaker\go.mod) | ForEach-Object { $_ -replace '// replace herofishingGoModule => ../herofishingGoModule // for local', 'replace herofishingGoModule => ../herofishingGoModule // for local' } | Set-Content matchmaker\go.mod"
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchmaker\go.mod) | ForEach-Object { $_ -replace 'replace herofishingGoModule => /go/src/herofishingGoModule // for docker', '// replace herofishingGoModule => /go/src/herofishingGoModule // for docker' } | Set-Content matchmaker\go.mod"