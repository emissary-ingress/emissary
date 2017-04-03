all: docker-images ambassador.yaml

VERSION=0.1.6

.ALWAYS:

ambassador.yaml: ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml > ambassador.yaml

docker-images: ambassador-image sds-image

ambassador-image: .ALWAYS
	docker build -t dwflynn/ambassador:$(VERSION) ambassador
#	docker push dwflynn/ambassador:$(VERSION)

sds-image: .ALWAYS
	docker build -t dwflynn/ambassador-sds:$(VERSION) sds
#	docker push dwflynn/ambassador-sds:$(VERSION)
	