# syntax = docker/dockerfile:1.3

###
# This dockerfile builds all the source code and docker images for the
# edge stack.
##

###
# How it works.
##
# All our source code is built in a single monolithic build
# container. (The "builder" stage below.) We then produce optimized
# release images by copying the specific binaries and entrypoints out
# of the build container. The build container must always contain all
# the binaries and entrypoints for any container we produce, and
# therefore (with the appropriate entrypoint) can function in place of
# any release container. The release containers are trivial stages
# that simply copy the relevant artifacts out of the builder
# container.

# This argument controls the base image that is used for the release
# containers.
ARG base="docker.io/frolvlad/alpine-glibc:alpine-3.12_glibc-2.32"

# This argument sets the base image for the build container image. This
# base image contains only third-party code.
ARG builderbase="i-forgot-to-set-build-arg-builderbase"

# This controls where we copy envoy from.
ARG envoy="i-forgot-to-set-build-arg-envoy"

########################################
# The builder image
########################################

FROM ${envoy} as envoy

FROM ${builderbase} as builder

WORKDIR /buildroot

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

# Some of the tests need redis-server to run
RUN apk add redis

COPY --from=envoy /usr/local/bin/envoy-static-stripped /usr/local/bin/envoy

ENV KUBECONFIG=/buildroot/kubeconfig.yaml

# XXX: this will go away
RUN mkdir -p /ambassador/sidecars && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py

RUN adduser dw --disabled-password
# SUDO_USERS HOSTS=(AS_USER) TAGS COMMANDS
RUN echo "dw ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/datawire
RUN chown dw /buildroot
USER dw

ENTRYPOINT [ "/bin/bash" ]

########################################
# The golang build stage
########################################

FROM ${builderbase} as golang

WORKDIR /go

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

# Build the golang binaries
ADD api api
ADD cmd cmd
ADD pkg pkg
ADD vendor vendor
ADD go.mod go.mod
ADD go.sum go.sum

RUN --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /go/bin && \
	time go build -mod=vendor -o /go/bin/ ./cmd/...

########################################
# The artifact build stage
########################################

FROM ${builderbase} as artifact-stage

WORKDIR /buildroot/ambassador

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

# Copy in the golang binaries
COPY --from=golang /go/bin/* /buildroot/bin/

# Setup the python code
ADD python python
RUN time pip install --no-deps -e python

# Run post-compile.sh
ADD build-aux build-aux
ADD post-compile.sh post-compile.sh
RUN bash post-compile.sh

# XXX: this will go away
RUN mkdir -p /ambassador/sidecars && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py

# These will be extracted into the optimized image later
ADD demo demo
ADD manifests/emissary/emissary-crds.yaml manifests/emissary/emissary-crds.yaml

########################################
# The optimized images
########################################

FROM artifact-stage as artifacts

FROM ${base} as ambassador

ARG py_version="i-forgot-to-set-build-arg-py-version"

# Always have an "ambassador" user as UID 8888. This is what we recommend
# people run as. (Note that the "-D" actually leaves the password locked.)
RUN adduser ambassador -u 8888 -G root -D -H -s /bin/false

# External stuff that should change infrequently
RUN apk --no-cache add bash curl python3=${py_version} libcap htop
RUN apk upgrade --no-cache
RUN ln -s /usr/bin/python3 /usr/bin/python
COPY --from=artifacts /usr/bin/kubectl /usr/bin/kubectl
COPY --from=artifacts /usr/lib/libyaml* /usr/lib/

# Other installers
COPY --from=artifacts /opt/image-build /opt/image-build
RUN /opt/image-build/install.sh

# External Python packages we use
COPY --from=artifacts /usr/lib/python3.8/site-packages /usr/lib/python3.8/site-packages

# Our envoy. The capabilities here grant the wrapper the ability to use the
# cap_net_bind_service cap and for Envoy to inherit it.
COPY --from=envoy /usr/local/bin/envoy-static-stripped /usr/local/bin/envoy
RUN setcap cap_net_bind_service=ei /usr/local/bin/envoy

# Our Go binaries. See envoy section for setcap info.
COPY --from=artifacts /opt/ambassador /opt/ambassador
RUN ln -s /opt/ambassador/bin/* /usr/local/bin/
RUN setcap cap_net_bind_service=p /opt/ambassador/bin/wrapper

# Our Python code
COPY --from=artifacts /buildroot/ambassador/python /buildroot/ambassador/python
RUN cd /buildroot/ambassador/python && python setup.py install

# Configuration, Docker demo stuff, the AES WebUI. The /ambassador bit changes
# in post-install so it's always stale. But it's pretty small, so it's not too
# bad to re-push every time.
COPY --from=artifacts /ambassador /ambassador
COPY --from=artifacts /buildroot/ambassador/demo/config /ambassador/ambassador-demo-config
COPY --from=artifacts /buildroot/ambassador/demo/services /ambassador/demo-services

# Fix permissions to allow correctly running as a non root user
# XXX: We could combine everything into one tree in the builder, fix permissions
# there, and then a use single COPY to get everything and avoid duplicating the
# (small amount of) data in a new layer for this RUN.
RUN chgrp -R 0 /ambassador && \
    chmod -R u+x /ambassador && \
    chmod -R g=u /ambassador /etc/passwd

WORKDIR /ambassador

# Force the HOME environment variable to a directory that'll always be writeable.
# We use /tmp/ambassador for this, and make sure it exists in our entrypoint,
# because trying to create it here in the Dockerfile doesn't always work very
# well in the face of situations like KAT volume-mounting /tmp/ambassador or
# the like.
ENV HOME=/tmp/ambassador

COPY --from=artifacts /bin/busybox /bin/busybox

ENTRYPOINT [ "bash", "/buildroot/ambassador/python/entrypoint.sh" ]

########################################

FROM ${base} as kat-client

COPY --from=artifacts /buildroot/bin/kat-client /usr/bin/kat-client
RUN mkdir /work && ln -s /usr/bin/kat-client /work/kat_client
CMD [ "sleep", "3600" ]

########################################

FROM ${base} as kat-server

COPY --from=artifacts /buildroot/bin/kat-server /usr/bin/kat-server

WORKDIR /work
COPY builder/server.crt server.crt
COPY builder/server.key server.key

ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api

EXPOSE 8080
CMD [ "kat-server" ]
