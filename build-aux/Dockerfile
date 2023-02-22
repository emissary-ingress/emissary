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
ARG base="i-forgot-to-set-build-arg-base"

# This argument sets the base image for the build container image. This
# base image contains only third-party code.
ARG builderbase="i-forgot-to-set-build-arg-builderbase"

# This controls where we copy envoy from.
ARG envoy="i-forgot-to-set-build-arg-envoy"

FROM ${envoy} as envoy

########################################
# The artifact build stage
########################################

FROM ${builderbase} as artifacts

# Setup the python code
ADD python /build/python
RUN cd /build/python && python setup.py install --install-scripts=/usr/local/bin && cd / && rm -rf /build

RUN ln -sf busyambassador /usr/local/bin/agent && \
    ln -sf busyambassador /usr/local/bin/apiext && \
    ln -sf busyambassador /usr/local/bin/kubestatus && \
    ln -sf busyambassador /usr/local/bin/watt

########################################
# The optimized images
########################################

FROM ${base} as ambassador

ARG py_version="i-forgot-to-set-build-arg-py-version"

# Always have an "ambassador" user as UID 8888. This is what we recommend
# people run as. (Note that the "-D" actually leaves the password locked.)
RUN adduser ambassador -u 8888 -G root -D -H -s /bin/false

# External stuff that should change infrequently
RUN apk --no-cache upgrade && \
    apk --no-cache add \
    bash \
    curl \
    python3=${py_version} \
    htop
RUN curl --fail -L https://storage.googleapis.com/kubernetes-release/release/v1.23.3/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && \
  chmod a+x /usr/local/bin/kubectl
COPY --from=artifacts /usr/lib/libyaml* /usr/lib/

# Python.
#
# Both our code and 3rd-party packages get installed to site-packages.
COPY --from=artifacts /usr/lib/python3.10/site-packages /usr/lib/python3.10/site-packages

# Scripts.
ADD python/entrypoint.sh \
    python/kubewatch.py \
    python/post_update.py \
    python/watch_hook.py \
    /usr/local/bin/

# Our envoy.
COPY --from=envoy /usr/local/bin/envoy-static-stripped /usr/local/bin/envoy

# Our execuables (Go, Python).
COPY --from=artifacts /usr/local/bin/* /usr/local/bin/

# setcap.
#
# The capabilities here grant the wrapper the ability to use the
# cap_net_bind_service cap and for Envoy to inherit it.
RUN apk --no-cache add libcap-utils && \
    setcap cap_net_bind_service=ei /usr/local/bin/envoy && \
    setcap cap_net_bind_service=p /usr/local/bin/capabilities_wrapper && \
    apk del libcap-utils

# Delete some things that cause problems for security scanners.
RUN python -m pip uninstall -y pip
RUN find / \( -iname '*sqlite*' -o -iname '*gdbm*' -o -iname '*smtplib*' -o -iname '*piptools*' -o -iname '*pip_tools*' -o -iname '*ensurepip*' \) -exec rm -rf -- '{}' \+

# Configuration, Docker demo stuff, the AES WebUI. The /ambassador bit changes
# in post-install so it's always stale. But it's pretty small, so it's not too
# bad to re-push every time.
COPY demo/config /ambassador/ambassador-demo-config
COPY demo/services /ambassador/demo-services

# Fix permissions to allow correctly running as a non root user
# XXX: We could combine everything into one tree in the builder, fix permissions
# there, and then a use single COPY to get everything and avoid duplicating the
# (small amount of) data in a new layer for this RUN.
RUN mkdir -p /ambassador/sidecars && \
    chgrp -R 0 /ambassador && \
    chmod -R u+x /ambassador && \
    chmod -R g=u /ambassador /etc/passwd

WORKDIR /ambassador

# Force the HOME environment variable to a directory that'll always be writeable.
# We use /tmp/ambassador for this, and make sure it exists in our entrypoint,
# because trying to create it here in the Dockerfile doesn't always work very
# well in the face of situations like KAT volume-mounting /tmp/ambassador or
# the like.
ENV HOME=/tmp/ambassador

ENTRYPOINT [ "bash", "entrypoint.sh" ]
