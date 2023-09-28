@echo off
REM 可在powershell中執行.\批次檔名稱.bat
REM 刪除namespace herofishing-game-server (此動作會一併移除該命名空間的所有資源，測試環境中使用)
REM 部屬完server後可以查看pod部屬狀況 kubectl get pods -n herofishing-game-server -o wide
REM 可以使用以下語法來查看特定pod上的log kubectl logs -f [POD_NAME] -n [NAMESPACE] (或直接透過gcp console介面來查看)
REM 取得遊戲server的ip與port kubectl get services -n herofishing-game-server  


@echo on

kubectl delete namespace herofishing-game-server
kubectl create namespace herofishing-game-server
kubectl apply -f K8s_Role.yaml
kubectl apply -f K8s_RoleBinding.yaml
kubectl apply -f Dev_Matchmaker.yaml

@echo off
REM 以下這行如果本來就有建立防火牆就不需要執行 可以註解掉否則會報錯誤
@echo on
gcloud compute firewall-rules create herofishing-matchmaker-firewall --allow tcp:32680 --target-tags herofishing-matchmaker --description "Firewall to allow Herofishing matchmaker TCP traffic"