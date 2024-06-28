########################################
# Multistage Dockerfile for Emissary
########################################

######## Envoy
ARG ENVOY_IMAGE=you-must-set-build-var-ENVOY-IMAGE

FROM ${ENVOY_IMAGE} as envoy-factory
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
FROM python:3.12.2-slim-bookworm as emissary-factory
# Repeat after every FROM line, sigh.
ARG TARGETARCH
ARG LIBARCH

# Based on https://github.com/alexdmoss/distroless-python...
RUN pip install --upgrade pip
WORKDIR /emissary
COPY python python
RUN pip install -r python/requirements.txt
RUN cd /emissary/python && python setup.py install

# Copy over ambassador.version
WORKDIR /buildroot/ambassador/python
COPY python/ambassador.version /buildroot/ambassador/python/ambassador.version

# Copy over busyambassador
WORKDIR /opt/ambassador/bin
COPY busyambassador busyambassador

# Make the symlinks we need
RUN for name in agent apiext entrypoint kubestatus; do ln -s busyambassador $name; done && \
    cd /usr/local/bin && \
    for name in agent apiext busyambassador entrypoint kubestatus; do ln -s ../../../opt/ambassador/bin/busyambassador $name; done

# Set up for a non-root environment.
# Always have an "ambassador" user as UID 8888. This is what we recommend
# people run as. (Note that the "-D" actually leaves the password locked.)
RUN adduser ambassador -u 8888 -G root -D -H -s /bin/false

# Fix permissions to allow correctly running as a non root user
# XXX: We could combine everything into one tree in the builder, fix permissions
# there, and then a use single COPY to get everything and avoid duplicating the
# (small amount of) data in a new layer for this RUN.
RUN install -d -g 0 -o 0 -m 0775 /ambassador
RUN chmod -R g=u /etc/passwd

# XXX: this will go away
RUN mkdir -p /ambassador/sidecars

######## Distroless final stage

FROM gcr.io/distroless/cc-debian12 as final
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
COPY --from=emissary-factory /usr/lib/${LIBARCH}-linux-gnu/libffi* /usr/lib/${LIBARCH}-linux-gnu/
COPY --from=emissary-factory /lib/${LIBARCH}-linux-gnu/libexpat* /lib/${LIBARCH}-linux-gnu/

# Things very helpful for debugging etc.
COPY --from=emissary-factory \
    /bin/echo /bin/ln /bin/rm /bin/sh /bin/chmod \
    /bin/

# Copy Emissary stuff
WORKDIR /opt/ambassador/bin
COPY --from=emissary-factory /opt/ambassador/bin/ /opt/ambassador/bin/

# Copy over emissary's Python artifacts
COPY --from=emissary-factory /usr/local/bin/* /usr/local/bin/
COPY --from=emissary-factory /usr/local/lib/python3.12/site-packages/ /usr/local/lib/python3.12/site-packages/
COPY --from=emissary-factory /ambassador /ambassador
COPY --from=emissary-factory /buildroot /buildroot

COPY --from=emissary-factory /emissary/python/ambassador.version /ambassador

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

ENTRYPOINT ["busyambassador", "entrypoint"]
