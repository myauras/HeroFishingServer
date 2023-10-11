gcloud config set project aurafortest
gcloud config set container/cluster cluster-gameserver
gcloud container clusters get-credentials cluster-gameserver --zone=asia-east1-c
kubectl config current-context