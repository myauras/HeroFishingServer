@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM 刪除namespace herofishing-service (此動作會一併移除該命名空間的所有資源，測試環境中使用)
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
kubectl apply -f Dev_Crontasker.yaml
@if ERRORLEVEL 1 exit /b 1