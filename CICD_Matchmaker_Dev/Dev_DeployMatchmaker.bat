@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM 部屬完server後可以查看pod部屬狀況 kubectl get pods -n herofishing-service -o wide
@REM 查看部屬的描述(如果部屬失敗可以用來查原因) kubectl describe pod game-server-deployment-30390-5ff599b69-6d7xn -n herofishing-service
REM 可以使用以下語法來查看特定pod上的log kubectl logs -f [POD_NAME] -n [NAMESPACE] (或直接透過gcp console介面來查看)
REM 取得遊戲server的ip與port kubectl get services -n herofishing-service  

@REM 如果k8s服務沒有啟動或沒有設定 會報錯誤Unable to connect to the server: dial tcp [::1]:8080: connectex: No connection could be made because the target machine actively refused it.
@REM 要使用以下指令來連接k8s與gke
@REM 先安裝gke工具 gcloud components install gke-gcloud-auth-plugin
@REM gcloud container clusters get-credentials YOUR_CLUSTER_NAME --zone YOUR_ZONE
@echo on
call Dev_SwitchProject.bat
kubectl apply -f Role.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f RoleBinding.yaml
@if ERRORLEVEL 1 exit /b 1
kubectl apply -f .\CICD_Matchmaker_Dev\Dev_Matchmaker.yaml
@if ERRORLEVEL 1 exit /b 1
@REM 建立k8s cluster的防火牆 以下這行如果本來就有建立防火牆就不需要執行 可以註解掉否則會報錯誤
@REM gcloud compute firewall-rules create herofishing-matchmaker-firewall-tcp --allow tcp:32680 --target-tags herofishing --description "Firewall to allow Herofishing matchmaker TCP traffic"

