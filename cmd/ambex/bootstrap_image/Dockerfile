ARG ENVOY_IMAGE
FROM envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab
WORKDIR /application
COPY bootstrap-ads.yaml bootstrap-ads.yaml
COPY ambex ambex
COPY example example
ENTRYPOINT ["envoy", "-l", "debug", "-c", "bootstrap-ads.yaml"]
