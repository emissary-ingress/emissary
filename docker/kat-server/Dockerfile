######## Distroless final stage

FROM docker.io/smallstep/step-cli:latest AS step-cli

WORKDIR /certs
# kat-server terminates TLS on 8443 with this cert, so it has to be a server
# leaf: `self-signed` sets Digital Signature key usage and serverAuth EKU,
# which Envoy's BoringSSL requires at handshake time regardless of whether the
# peer verifies the chain. `--subtle` is what step wants before it will issue a
# self-signed leaf, and `--not-after` is needed because that profile defaults
# to a 24-hour lifetime.
RUN step certificate create kat.server.local /certs/server.crt /certs/server.key \
    --profile self-signed --subtle --san kat.server.local \
    --not-after 87600h --no-password --insecure

FROM gcr.io/distroless/cc-debian12 as final
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# This is associated with the emissary repo.
LABEL org.opencontainers.image.source=https://github.com/emissary-ingress/emissary

ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api

WORKDIR /work
COPY kat-server kat-server
COPY --from=step-cli /certs/server.crt /certs/server.key /work/

ENTRYPOINT ["kat-server"]
