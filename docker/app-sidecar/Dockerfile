FROM envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab

# Files that do NOT have the version number embedded
COPY ./entrypoint.sh ./ambex ./bootstrap-ads.yaml /application/
COPY ./data/* /application/data/

# Files that DO have the version number embedded
COPY ./app-sidecar /application/

WORKDIR /application
CMD [ "/application/entrypoint.sh" ]
