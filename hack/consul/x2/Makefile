.PHONY: cert ambassador-cert

cluster:
	KUBECONFIG=$(PWD)/ambassador-940.yaml \
	gcloud container clusters create ambasssador-940 \
      --cluster-version 1.11.2-gke.18 \
      --project datawireio

cluster-delete:
	gcloud container clusters delete ambassador-940

cert: HOST=localhost
cert:
	openssl req \
	-newkey rsa:2048 \
	-nodes \
	-keyout insecure.key \
	-x509 \
	-days 365 \
	-out insecure.crt \
	-subj '/CN=$(HOST)'

ambassador-certs: cert
	kubectl create secret tls ambassador-certs \
		--cert=insecure.crt \
		--key=insecure.key \
		--dry-run \
		-o yaml > ambassador/secret.ambassador-certs.yaml