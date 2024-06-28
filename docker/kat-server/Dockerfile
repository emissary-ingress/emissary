######## Distroless final stage

FROM gcr.io/distroless/cc-debian12 as final
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# This is associated with the emissary repo.
LABEL org.opencontainers.image.source=https://github.com/emissary-ingress/emissary

ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api

WORKDIR /work
COPY kat-client kat-client
COPY docker/kat-server/server.crt docker/kat-server/server.key /work

ENTRYPOINT ["kat-server"]
