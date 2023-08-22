kubectl delete namespace herofishing-game-server
kubectl create namespace herofishing-game-server
kubectl apply -f K8s_Role.yaml
kubectl apply -f K8s_RoleBinding.yaml
kubectl apply -f Dev_Matchmaker.yaml

gcloud compute firewall-rules create herofishing-matchmaker-firewall \
  --allow tcp:32680 \
  --target-tags herofishing-matchmaker \
  --description "Firewall to allow Herofishing matchmaker TCP traffic"