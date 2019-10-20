# arguments
ARG artifacts
ARG aoss
FROM ${artifacts} as artifacts
FROM ${aoss} as aoss

# main
FROM aoss
COPY --from=artifacts /buildroot/bin/amb-sidecar /ambassador/sidecars/amb-sidecar
