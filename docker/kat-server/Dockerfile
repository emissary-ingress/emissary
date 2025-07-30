######## Distroless final stage

FROM docker.io/smallstep/step-cli:latest AS step-cli

WORKDIR /certs
RUN step certificate create kat.server.local /certs/server.crt /certs/server.key --profile root-ca --no-password --insecure

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
COPY --from=step-cli /certs/server.crt /certs/server.key /work/

ENTRYPOINT ["kat-server"]
