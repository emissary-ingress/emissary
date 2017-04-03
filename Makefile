all: docker-images ambassador.yaml

VERSION=0.1.9

.ALWAYS:

ambassador.yaml: ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml > ambassador.yaml

docker-images: ambassador-image sds-image

ambassador-image: .ALWAYS
	docker build -t dwflynn/ambassador:$(VERSION) ambassador
	if [ -n "$(DOCKER_REGISTRY)" ]; then \
		docker push $(DOCKER_REGISTRY)/ambassador:$(VERSION); \
	fi

sds-image: .ALWAYS
	docker build -t dwflynn/ambassador-sds:$(VERSION) sds
	if [ -n "$(DOCKER_REGISTRY)" ]; then \
		docker push $(DOCKER_REGISTRY)/ambassador-sds:$(VERSION); \
	fi
	