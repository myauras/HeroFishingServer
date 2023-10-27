@REM 部屬完server後可以查看pod部屬狀況 kubectl get pods -n herofishing-gameserver -o wide
@REM 查看部屬的描述(如果部屬失敗可以用來查原因) kubectl describe gameserver herofishing-matchgame -n herofishing-gameserver

call Dev_SwitchProject.bat
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Dev_fleet.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f Dev_fleetautoscaler.yaml
@if ERRORLEVEL 1 exit /b 1

@REM 建立k8s cluster的防火牆 以下這行如果本來就有建立防火牆就不需要執行 可以註解掉否則會報錯誤
@REM gcloud compute firewall-rules create herofishing-matchgame-firewall-udp --allow udp:7000-8000 --target-tags herofishing --description "Herofishing firewall to allow game server udp traffic"
@REM gcloud compute firewall-rules create herofishing-matchgame-firewall-tcp --allow tcp:7000-8000 --target-tags herofishing --description "Herofishing firewall to allow game server tcp traffic"
