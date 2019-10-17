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

# This argument controlls the base image is used for both our build
# container and for the release containers.
ARG base="frolvlad/alpine-glibc:alpine-3.10"

# This argument controls where the release images get their artifacts
# from. We override it for incremental builds, but default it to the
# builder stage so this Dockerfile works out of the box.
ARG artifacts="builder"

# This controls where we copy envoy from.
ARG envoy="quay.io/datawire/ambassador:0.83.0"

########################################
# The builder image
########################################

FROM ${envoy} as envoy
FROM ${base} as builder

WORKDIR /buildroot

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

# The YAML parser is... special. To get the C version, we need to install Cython and libyaml, then
# build it locally -- just using pip won't work.

RUN apk --no-cache add bash curl jq rsync python3 python3-dev build-base libffi-dev openssl-dev sudo \
                       iptables docker openssh-client libcap yaml-dev cython && \
    pip3 install -U pip && \
    curl --fail https://dl.google.com/go/go1.13.linux-amd64.tar.gz | tar -C /usr/local -xzf - && \
    curl --fail -L https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl -o /usr/bin/kubectl && \
    chmod a+x /usr/bin/kubectl && \
    ln -s /usr/bin/python3 /usr/bin/python

RUN chmod u+s $(which docker)

COPY --from=envoy /usr/local/bin/envoy /usr/local/bin/envoy

ENV KUBECONFIG=/buildroot/kubeconfig.yaml

# Download, build, and install PyYAML.
RUN mkdir /tmp/pyyaml && \
    cd /tmp/pyyaml && \
    curl -O -L http://pyyaml.org/download/pyyaml/PyYAML-5.1.tar.gz && \
    tar xzf PyYAML-5.1.tar.gz && \
    cd PyYAML-5.1 && \
    python3 setup.py --with-libyaml install

# Then we can do the rest of the Python stuff.
COPY requirements.txt .
RUN pip install -r requirements.txt

# XXX: this will go away
RUN mkdir /ambassador && \
    ln -s /buildroot/bin/watt /ambassador/watt && \
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
# The optimized images
########################################

FROM ${artifacts} as artifacts

FROM ${base} as ambassador

RUN apk --no-cache add bash curl python3 libcap

RUN ln -s /usr/bin/python3 /usr/bin/python

COPY --from=envoy /usr/local/bin/envoy /usr/local/bin/envoy
COPY --from=builder /usr/lib/python3.7/site-packages /usr/lib/python3.7/site-packages
COPY --from=builder /usr/lib/libyaml* /usr/lib/

COPY --from=artifacts /buildroot/bin/ambex /usr/bin/ambex
COPY --from=artifacts /buildroot/bin/watt /usr/bin/watt
COPY --from=artifacts /buildroot/bin/kubestatus /usr/bin/kubestatus
COPY --from=artifacts /usr/bin/kubectl /usr/bin/kubectl
COPY --from=artifacts /buildroot/ambassador/python /buildroot/ambassador/python
RUN cd /buildroot/ambassador/python && python setup.py install

# XXX: this will go away
RUN mkdir /ambassador && \
    ln -s /usr/bin/watt /ambassador/watt && \
    ln -s /usr/bin/kubestatus /ambassador/kubestatus && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py

COPY --from=artifacts /buildroot/ambassador/demo/config /ambassador/ambassador-demo-config
COPY --from=artifacts /buildroot/ambassador/demo/services /ambassador/demo-services

WORKDIR /ambassador

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
COPY server.crt server.crt
COPY server.key server.key

ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api

EXPOSE 8080
CMD [ "kat-server" ]
