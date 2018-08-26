ARG ENVOY_IMAGE
FROM ${ENVOY_IMAGE}
WORKDIR /application
COPY bootstrap-ads.yaml bootstrap-ads.yaml
ENTRYPOINT ["envoy", "-l", "debug", "-c", "bootstrap-ads.yaml"]
