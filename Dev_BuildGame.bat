@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM Build Image並推上google artifact registry, google放image的地方)

REM 如果puch image發生錯誤可以跑以下重新登入跟認證流程試試看
@REM gcloud auth revoke
@REM gcloud auth login
@REM docker logout asia-east1-docker.pkg.dev
@REM gcloud auth configure-docker asia-east1-docker.pkg.dev

@echo on


REM =======change go.mod for docker setting=======
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchgame\go.mod) | ForEach-Object { $_ -replace 'replace herofishingGoModule => ../herofishingGoModule // for local', '// replace herofishingGoModule => ../herofishingGoModule // for local' } | Set-Content matchgame\go.mod"
@if ERRORLEVEL 1 exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchgame\go.mod) | ForEach-Object { $_ -replace '// replace herofishingGoModule => /home/herofishingGoModule // for docker', 'replace herofishingGoModule => /home/herofishingGoModule // for docker' } | Set-Content matchgame\go.mod"
@if ERRORLEVEL 1 exit /b 1

REM =======build image=======
docker build --no-cache -f matchgame/Dockerfile -t asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchgame .
@if ERRORLEVEL 1 exit /b 1

REM =======push image=======
docker push asia-east1-docker.pkg.dev/aurafortest/herofishing/herofishing-matchgame
@if ERRORLEVEL 1 exit /b 1

REM =======change go.mod back to local setting=======
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchgame\go.mod) | ForEach-Object { $_ -replace '// replace herofishingGoModule => ../herofishingGoModule // for local', 'replace herofishingGoModule => ../herofishingGoModule // for local' } | Set-Content matchgame\go.mod"
@if ERRORLEVEL 1 exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content matchgame\go.mod) | ForEach-Object { $_ -replace 'replace herofishingGoModule => /home/herofishingGoModule // for docker', '// replace herofishingGoModule => /home/herofishingGoModule // for docker' } | Set-Content matchgame\go.mod"
@if ERRORLEVEL 1 exit /b 1