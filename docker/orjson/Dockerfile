# syntax=docker/dockerfile:1.3-labs

FROM docker.io/frolvlad/alpine-glibc:alpine-3.15

# Install dependencies for building orjson
RUN apk --no-cache add \
    cargo \
    patchelf \
    py3-pip=~20.3.4 \
    python3=~3.9.7 \
    python3-dev

# orjson's `build.rs` file hard-codes "python" as the program name.
RUN ln -s /usr/bin/python3 /usr/local/bin/python

RUN adduser -D builduser
USER builduser
WORKDIR /home/builduser/workdir

RUN pip3 wheel orjson
