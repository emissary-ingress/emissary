# syntax = docker/dockerfile:1.3-labs

FROM docker.io/frolvlad/alpine-glibc:alpine-3.15 as base

RUN apk --no-cache add \
    bash \
    curl \
    python3=~3.9.7 \
    py3-pip=~20.3.4 \
    py3-yaml

################################################################################
# Get+patch+build the pkgname=python3 package sources                          #
################################################################################
# We want our 'python3' package to use glibc, not musl.  So grab
# Alpine's sources for the 'python3' package, patch it to use glibc,
# and then build it.
RUN --mount=type=cache,target=/home/builduser/.ccache <<EOC
    set -e

FROM base as orjson

RUN apk --no-cache add \
    rust \
    cargo \
    patchelf \
RUN ln -s /usr/bin/python3 /usr/bin/python
    RUN pip3 wheel orjson==3.6.6

FROM base

COPY --from=orjson orjson.whl
RUN pip3 install orjson.whl
