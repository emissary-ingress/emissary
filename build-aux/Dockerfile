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
ADD python /buildroot/ambassador/python
RUN time pip install --no-deps -e /buildroot/ambassador/python

# Set up /opt/ambassador/bin/ (for the production image)
RUN sudo install -D -t /opt/ambassador/bin/ /usr/local/bin/busyambassador && \
    sudo ln -sf busyambassador /opt/ambassador/bin/agent && \
    sudo ln -sf busyambassador /opt/ambassador/bin/apiext && \
    sudo ln -sf busyambassador /opt/ambassador/bin/kubestatus && \
    sudo ln -sf busyambassador /opt/ambassador/bin/watt && \
    sudo install /usr/local/bin/capabilities_wrapper /opt/ambassador/bin/wrapper

# XXX: this will go away
RUN mkdir -p /ambassador/sidecars && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py

# These will be extracted into the optimized image later
ADD demo /buildroot/ambassador/demo
ADD manifests/emissary/emissary-crds.yaml.in /buildroot/ambassador/manifests/emissary/emissary-crds.yaml

########################################
# The optimized images
########################################

FROM ${base} as ambassador

ARG py_version="i-forgot-to-set-build-arg-py-version"

# Always have an "ambassador" user as UID 8888. This is what we recommend
# people run as. (Note that the "-D" actually leaves the password locked.)
RUN adduser ambassador -u 8888 -G root -D -H -s /bin/false

# External stuff that should change infrequently
RUN apk --no-cache add bash curl python3=${py_version} libcap htop
RUN apk upgrade --no-cache
COPY --from=artifacts /usr/bin/kubectl /usr/bin/kubectl
COPY --from=artifacts /usr/lib/libyaml* /usr/lib/

# External Python packages we use
COPY --from=artifacts /usr/lib/python3.10/site-packages /usr/lib/python3.10/site-packages

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

# Delete some things that cause problems for security scanners.
RUN python -m pip uninstall -y pip
RUN find / \( -iname '*sqlite*' -o -iname '*gdbm*' -o -iname '*smtplib*' -o -iname '*piptools*' -o -iname '*pip_tools*' -o -iname '*ensurepip*' \) -exec rm -rf -- '{}' \+

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

ENTRYPOINT [ "bash", "/buildroot/ambassador/python/entrypoint.sh" ]
