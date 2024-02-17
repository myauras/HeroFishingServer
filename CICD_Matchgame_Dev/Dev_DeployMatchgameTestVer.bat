@echo off
REM 不使用Agones的部屬, 用於個人開發測試用, 不會影響到其他人測試
@REM 查看部屬的描述(如果部屬失敗可以用來查原因) kubectl describe pod herofishing-matchgame-testver-6848d75d5c-b288c -n herofishing-gameserver
@echo on
call Dev_SwitchProject.bat
kubectl apply -f .\CICD_Matchgame_Dev\Role.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f .\CICD_Matchgame_Dev\RoleBinding.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f .\CICD_Matchgame_Dev\Dev_MatchgameTestVer.yaml
@if ERRORLEVEL 1 exit /b 1