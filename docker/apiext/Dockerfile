######## Distroless final stage

FROM gcr.io/distroless/cc-debian12 as final
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# This is associated with the emissary repo.
LABEL org.opencontainers.image.source=https://github.com/emissary-ingress/emissary

WORKDIR /usr/local/bin
COPY apiext apiext
USER 8888:8888

ENTRYPOINT ["apiext", "emissary-apiext"]
