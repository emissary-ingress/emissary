FROM envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab

# Files that do NOT have the version number embedded
COPY ./ambex /opt/ambassador-pro/
COPY ./data/* /opt/ambassador-pro/data/

# Files that DO have the version number embedded
COPY ./app-sidecar /opt/ambassador-pro/

CMD [ "/opt/ambassador-pro/app-sidecar" ]
