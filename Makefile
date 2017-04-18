all: docker-images ambassador.yaml statsd-sink.yaml

VERSION=0.5.0

.ALWAYS:

ambassador-sds.yaml: .ALWAYS
	sh templates/ambassador-sds.yaml.sh > ambassador-sds.yaml

ambassador-rest.yaml: .ALWAYS
	sh templates/ambassador-rest.yaml.sh > ambassador-rest.yaml

ambassador.yaml: ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml > ambassador.yaml

statsd-sink.yaml: .ALWAYS
	sh templates/statsd-sink.yaml.sh > statsd-sink.yaml

docker-images: ambassador-image sds-image statsd-image prom-statsd-exporter

ambassador-image: .ALWAYS
	scripts/docker_build_maybe_push dwflynn ambassador $(VERSION) ambassador

sds-image: .ALWAYS
	scripts/docker_build_maybe_push dwflynn ambassador-sds $(VERSION) sds

statsd-image: .ALWAYS
	scripts/docker_build_maybe_push ark3 statsd $(VERSION) statsd

prom-statsd-exporter: .ALWAYS
	scripts/docker_build_maybe_push ark3 prom-statsd-exporter $(VERSION) prom-statsd-exporter
