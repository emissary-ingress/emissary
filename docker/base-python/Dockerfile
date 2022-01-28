###
# This dockerfile builds the base image for the builder container. See
# the main Dockerfile for more information about what the builder
# container is and how code in this repo is built.
#
# Originally this base was built as part of the builder container's
# bootstrap process. We discovered that minor network interruptions
# would break these steps, and such interruptions were common on our
# cloud CI system. We decided to separate out these steps so that any
# one of them is much less likely to be the cause of a network-related
# failure, i.e. a flake.
#
# See the comment before the build_builder_base() function in builder.sh
# to see when and how often this base image is built and pushed.
##

########################################
# Third-party code
########################################

FROM docker.io/frolvlad/alpine-glibc:alpine-3.15

WORKDIR /buildroot

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

RUN apk --no-cache add \
    bash \
    gcc \
    make \
    musl-dev \
    curl \
    cython \
    docker-cli \
    git \
    iptables \
    jq \
    libcap \
    libcap-dev \
    libffi-dev \
    ncurses \
    openssl-dev \
    py3-pip=~20.3.4 \
    python3=~3.9.7 \
    python3-dev \
    rust \
    cargo \
    patchelf \
    rsync \
    sudo \
    yaml-dev \
    && ln -s /usr/bin/python3 /usr/bin/python \
    && chmod u+s $(which docker)

# Consult
# https://github.com/jazzband/pip-tools/#versions-and-compatibility to
# select a pip-tools version that corresponds to the 'py3-pip' and
# 'python3' versions above.
RUN pip3 install pip-tools==6.3.1

RUN curl --fail -L https://dl.google.com/go/go1.17.6.linux-amd64.tar.gz | tar -C /usr/local -xzf -

RUN curl --fail -L https://storage.googleapis.com/kubernetes-release/release/v1.22.2/bin/linux/amd64/kubectl -o /usr/bin/kubectl && \
    chmod a+x /usr/bin/kubectl

# The YAML parser is... special. To get the C version, we need to install Cython and libyaml, then
# build it locally -- just using pip won't work.
#
# Download, build, and install PyYAML.
RUN mkdir /tmp/pyyaml && \
  cd /tmp/pyyaml && \
  curl -o pyyaml-5.4.1.1.tar.gz -L https://github.com/yaml/pyyaml/archive/refs/tags/5.4.1.1.tar.gz && \
  tar xzf pyyaml-5.4.1.1.tar.gz && \
  cd pyyaml-5.4.1.1 && \
  python3 setup.py --with-libyaml install

# orjson is also special.  The wheels on PyPI rely on glibc, so we
# need to use cargo/rustc/patchelf to build a musl-compatible version.
RUN pip3 install orjson==3.6.6
