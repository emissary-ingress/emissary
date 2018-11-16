cluster:
	KUBECONFIG=$(PWD)/ambassador-940.yaml \
	gcloud container clusters create ambasssador-940 \
      --cluster-version 1.11.2-gke.18 \
      --project datawireio

cluster-delete:
	gcloud container clusters delete ambassador-940