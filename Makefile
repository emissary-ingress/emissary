all: docker-images ambassador.yaml

.ALWAYS:

ambassador.yaml: ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml > ambassador.yaml

docker-images: ambassador-image sds-image

ambassador-image: .ALWAYS
	docker build -t dwflynn/ambassador:0.1.6 ambassador
	docker push dwflynn/ambassador:0.1.6

sds-image: .ALWAYS
	docker build -t dwflynn/ambassador-sds:0.1.6 sds
	docker push dwflynn/ambassador-sds:0.1.6
	