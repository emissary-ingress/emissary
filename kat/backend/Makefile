TAG=5

build:
	docker build . -t quay.io/datawire/kat-backend:${TAG}

backend: build
	docker push quay.io/datawire/kat-backend:${TAG}
