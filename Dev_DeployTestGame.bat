@REM 部屬完server後可以查看pod部屬狀況 kubectl get pods -n test-gameserver -o wide

call Dev_SwitchProject.bat
@if ERRORLEVEL 1 exit /b 1
kubectl delete namespace test-gameserver
@if ERRORLEVEL 1 exit /b 1
kubectl create namespace test-gameserver
kubectl apply -f Role.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Matchgame_RoleBinding.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Dev_DeployTestGame.yaml
@if ERRORLEVEL 1 exit /b 1
