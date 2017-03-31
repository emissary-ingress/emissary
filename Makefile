all: ambassador

ambassador: docker-image

DOCKER_SOURCES = \
    Dockerfile \
    \
    ambassador.py \
    envoy-template.json \
    requirements.txt \
    \
    entrypoint.sh \
    envoy-restarter.py \
    envoy-wrapper.sh

docker-image: $(DOCKER_SOURCES)
	docker build -t datawire/ambassador:0.1.0 .
