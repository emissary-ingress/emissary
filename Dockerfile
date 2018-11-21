ARG ENVOY_IMAGE
FROM ${ENVOY_IMAGE}
WORKDIR /application
COPY bootstrap-ads.yaml bootstrap-ads.yaml
COPY ambex_for_image ambex
COPY example example
ENTRYPOINT ["envoy", "-l", "debug", "-c", "bootstrap-ads.yaml"]
