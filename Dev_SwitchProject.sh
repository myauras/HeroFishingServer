gcloud config set project aurafortest
gcloud config set container/cluster gameserver-cluster
gcloud container clusters get-credentials gameserver-cluster --zone=asia-east1-a
kubectl config current-context