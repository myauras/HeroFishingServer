@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM Build Image並推上google artifact registry, google放image的地方)

REM 如果puch image發生錯誤可以跑以下重新登入跟認證流程試試看
@REM gcloud auth revoke
@REM gcloud auth login
@REM docker logout asia-east1-docker.pkg.dev
@REM gcloud auth configure-docker asia-east1-docker.pkg.dev

@echo on

REM =======Check docker is running=======
@echo off
tasklist /FI "IMAGENAME eq Docker Desktop.exe" 2>NUL | find /I /N "Docker Desktop.exe">NUL
if "%ERRORLEVEL%"=="0" (
    echo Docker Desktop is Running
) else (
    echo Docker Desktop isn't Running
    exit
)
@echo on

REM =======Change go.mod for docker setting=======
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content lobby\go.mod) | ForEach-Object { $_ -replace 'replace herofishingGoModule => ../herofishingGoModule // for local', '// replace herofishingGoModule => ../herofishingGoModule // for local' } | Set-Content lobby\go.mod"
@if ERRORLEVEL 1 exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content lobby\go.mod) | ForEach-Object { $_ -replace '// replace herofishingGoModule => /home/herofishingGoModule // for docker', 'replace herofishingGoModule => /home/herofishingGoModule // for docker' } | Set-Content lobby\go.mod"
@if ERRORLEVEL 1 exit /b 1

REM =======Build image=======
docker build --no-cache -f lobby/Dockerfile -t asia-east1-docker.pkg.dev/fourth-waters-410202/herofishing/herofishing-lobby:0.1.31 .
@if ERRORLEVEL 1 exit /b 1

REM =======Push image=======
docker push asia-east1-docker.pkg.dev/fourth-waters-410202/herofishing/herofishing-lobby:0.1.31
@if ERRORLEVEL 1 exit /b 1

REM =======Change go.mod back to local setting=======
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content lobby\go.mod) | ForEach-Object { $_ -replace '// replace herofishingGoModule => ../herofishingGoModule // for local', 'replace herofishingGoModule => ../herofishingGoModule // for local' } | Set-Content lobby\go.mod"
@if ERRORLEVEL 1 exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -command "(Get-Content lobby\go.mod) | ForEach-Object { $_ -replace 'replace herofishingGoModule => /home/herofishingGoModule // for docker', '// replace herofishingGoModule => /home/herofishingGoModule // for docker' } | Set-Content lobby\go.mod"
@if ERRORLEVEL 1 exit /b 1