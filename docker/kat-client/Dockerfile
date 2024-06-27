######## Distroless final stage

FROM gcr.io/distroless/cc-debian12 as final
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# This is associated with the emissary repo.
LABEL org.opencontainers.image.source=https://github.com/emissary-ingress/emissary

WORKDIR /work
COPY kat-client kat_client
USER 0:0

ENTRYPOINT ["sh", "-c", "while true; do sleep 3600; done"]
