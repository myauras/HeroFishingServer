@echo off

REM 刪除namespace herofishing-game-server (此動作會一併移除該命名空間的所有資源，測試環境中使用)

@echo on

kubectl delete namespace herofishing-game-server
kubectl create namespace herofishing-game-server
kubectl apply -f K8s_Role.yaml
kubectl apply -f K8s_RoleBinding.yaml
kubectl apply -f Dev_Matchmaker.yaml

gcloud compute firewall-rules create herofishing-matchmaker-firewall --allow tcp:32680 --target-tags herofishing-matchmaker --description "Firewall to allow Herofishing matchmaker TCP traffic"