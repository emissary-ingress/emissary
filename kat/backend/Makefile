TAG := 8

build:
	docker build . -t quay.io/datawire/kat-backend:${TAG}

backend: xds build
	docker push quay.io/datawire/kat-backend:${TAG}
