########################################
# Multistage Dockerfile for Emissary
########################################

######## Envoy
ARG ENVOY_IMAGE=you-must-set-build-var-ENVOY-IMAGE

FROM ${ENVOY_IMAGE} AS envoy-factory
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

######## Python

# We use the Python 3.12.2-slim-bookworm image to manage Emissary's
# Python half, then replicate some of the Google distroless Python image
# ourselves, based on https://github.com/alexdmoss/distroless-python
# (we'd use that directly, but its published images aren't multi-arch).
#
# We don't use Google's distroless image directly because its Python
# version is older than we want.
FROM docker.io/library/python:3.12.2-slim-bookworm AS emissary-factory
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# Based on https://github.com/alexdmoss/distroless-python...
RUN pip install --upgrade pip uv

# Copy over ambassador.version
WORKDIR /buildroot/ambassador/python
COPY python/ambassador.version /buildroot/ambassador/python/ambassador.version

# Set up the Python environment that Emissary needs.
WORKDIR /emissary
COPY python /emissary/python
COPY pyproject.toml /emissary/pyproject.toml
COPY LICENSE /emissary/LICENSE

# Have uv install a Python for us.
ENV UV_PYTHON_INSTALL_DIR=/opt/ambassador/python
RUN uv python install 3.12.2

# Set up /opt/ambassador as a virtual environment for Emissary...
RUN uv venv /opt/ambassador/emissary --python 3.12.2

# Build wheels for ambassador and ambassador-diag...
RUN . /opt/ambassador/emissary/bin/activate && \
    cd /emissary/python/ambassador && uv build --wheel
RUN . /opt/ambassador/emissary/bin/activate && \
    cd /emissary/python/ambassador-diag && uv build --wheel

# ...then use uv pip install to install our wheels.
RUN cd /opt/ambassador/emissary && \
    uv pip install \
    /emissary/dist/ambassador-4.0.0.dev0-py2.py3-none-any.whl \
    /emissary/dist/ambassador_diag-4.0.0.dev0-py2.py3-none-any.whl

# Copy over busyambassador
WORKDIR /opt/ambassador/bin
COPY busyambassador busyambassador

# Make the symlinks we need
RUN for name in agent apiext entrypoint kubestatus; do ln -s busyambassador $name; done
RUN ln -s ../emissary/bin/diagd diagd

# Set up for a non-root environment.
# Always have an "ambassador" user as UID 8888. This is what we recommend
# people run as. (Note that the "-D" actually leaves the password locked.)
RUN adduser ambassador \
    --uid 8888 \
    --ingroup root \
    --disabled-login \
    --no-create-home \
    --shell /bin/false \
    --comment ambassador

# Fix permissions to allow correctly running as a non root user
# XXX: We could combine everything into one tree in the builder, fix permissions
# there, and then a use single COPY to get everything and avoid duplicating the
# (small amount of) data in a new layer for this RUN.
RUN install -d -g 0 -o 0 -m 0775 /ambassador
RUN chmod -R g=u /etc/passwd

# XXX: this will go away
RUN mkdir -p /ambassador/sidecars

######## Distroless final stage

FROM gcr.io/distroless/cc-debian12 AS final
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# This is associated with the emissary repo.
LABEL org.opencontainers.image.source=https://github.com/emissary-ingress/emissary

# Copy over the stripped envoy binary
WORKDIR /usr/local/bin
COPY --from=envoy-factory /usr/local/bin/envoy /usr/local/bin/envoy

# from https://github.com/alexdmoss/distroless-python
# Copy Python itself
COPY --from=emissary-factory /usr/local/lib/ /usr/local/lib/
COPY --from=emissary-factory /usr/local/bin/python /usr/local/bin/
COPY --from=emissary-factory /etc/ld.so.cache /etc/
COPY --from=emissary-factory /lib/${LIBARCH}-linux-gnu/libz.so.1 /lib/${LIBARCH}-linux-gnu/
COPY --from=emissary-factory /lib/${LIBARCH}-linux-gnu/libselinux* /lib/${LIBARCH}-linux-gnu/
COPY --from=emissary-factory /lib/${LIBARCH}-linux-gnu/libpcre* /lib/${LIBARCH}-linux-gnu/
COPY --from=emissary-factory /usr/lib/${LIBARCH}-linux-gnu/libffi* /usr/lib/${LIBARCH}-linux-gnu/
COPY --from=emissary-factory /lib/${LIBARCH}-linux-gnu/libexpat* /lib/${LIBARCH}-linux-gnu/

# For bash
COPY --from=emissary-factory /lib/${LIBARCH}-linux-gnu/libtinfo* /lib/${LIBARCH}-linux-gnu/
COPY --from=emissary-factory /usr/bin/bash /usr/bin/bash

# Things very helpful for debugging etc.
COPY --from=emissary-factory \
    /bin/cat /bin/chmod /bin/echo /bin/env /bin/find \
    /bin/grep /bin/id /bin/ln /bin/ls /bin/mkdir \
    /bin/pwd /bin/rm /bin/sh \
    /bin/

# Copy Emissary stuff
WORKDIR /opt/ambassador
COPY --from=emissary-factory /opt/ambassador /opt/ambassador

# Temporary hack
COPY --from=emissary-factory /emissary /ffs
COPY --from=emissary-factory /ambassador /ambassador
COPY --from=emissary-factory /buildroot /buildroot

# The copy in /buildroot is enough.
# COPY --from=emissary-factory /emissary/python/ambassador.version /ambassador

# Copy /etc/passwd so we have our user.
COPY --from=emissary-factory /etc/passwd /etc/passwd

# Force permissions to allow correctly running as a non root user
RUN chmod -R g=u /ambassador /etc/passwd

# Force the HOME environment variable to a directory that'll always be writeable.
# We use /tmp/ambassador for this, and make sure it exists in our entrypoint,
# because trying to create it here in the Dockerfile doesn't always work very
# well in the face of situations like KAT volume-mounting /tmp/ambassador or
# the like.
ENV HOME=/tmp/ambassador
ENV PATH=/usr/local/bin:$PATH

# Set up symlinks in /usr/local/bin for busyambassador and friends.
RUN for name in agent apiext busyambassador diagd entrypoint kubestatus; do \
    ln -s ../../../opt/ambassador/bin/$name /usr/local/bin/$name; \
    done

ENTRYPOINT ["busyambassador", "entrypoint"]
