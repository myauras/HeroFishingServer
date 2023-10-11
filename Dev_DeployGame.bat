@REM 部屬完server後可以查看pod部屬狀況 kubectl get pods -n herofishing-gameserver -o wide

call Dev_SwitchProject.bat
@if ERRORLEVEL 1 exit /b 1
kubectl delete namespace herofishing-gameserver
@if ERRORLEVEL 1 exit /b 1
kubectl create namespace herofishing-gameserver
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Role.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Matchmaker_RoleBinding.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Dev_fleet.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Dev_fleetautoscaler.yaml
@if ERRORLEVEL 1 exit /b 1
