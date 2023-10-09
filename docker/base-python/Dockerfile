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

FROM docker.io/frolvlad/alpine-glibc:alpine-3.17_glibc-2.34

WORKDIR /buildroot

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

# For the most-part, you should not pin specific versions in this
# `apk` command; it will always choose the latest version being
# shipped by pinned version of Alpine.  This will automatically pull
# in any patches, but not breaking changes.
#
# Because of how we cache the base image
# (`docker/base-python.docker.gen`), it'll keep selected versions
# around for a week.  If there's a security patch that we want to pull
# in without waiting a week for the cache to roll over, instead of
# fussing with pinning a specific package version, simply make an
# inconsequential change that changes the file's checksum (such as
# bumping this comment):
#
#   forced-bumps: 0
#
# The exception to this is that we pin 'python3' and 'py3-pip' because
# the 'pip-tools' version below will also need to change if those
# versions change.  And even then, don't get more precise with the
# version number than we need to ensure that the pip-tools version
# agrees.
RUN apk --no-cache add \
  bash \
  gcc \
  make \
  musl-dev \
  curl \
  docker-cli \
  git \
  iptables \
  jq \
  libcap \
  libcap-dev \
  libffi-dev \
  ncurses \
  openssl-dev \
  py3-pip=~22 \
  python3=~3.10 \
  python3-dev \
  rust \
  cargo \
  patchelf \
  rsync \
  sudo \
  yaml-dev \
  && chmod u+s $(which docker)

# Consult
# https://github.com/jazzband/pip-tools/#versions-and-compatibility to
# select a pip-tools version that corresponds to the 'py3-pip' and
# 'python3' versions above.
RUN pip3 install "Cython<3.0" pip-tools==7.3

RUN curl --fail -L https://dl.google.com/go/go1.20.9.linux-amd64.tar.gz | tar -C /usr/local -xzf -

# The YAML parser is... special. To get the C version, we need to install Cython and libyaml, then
# build it locally -- just using pip won't work.
#
# Download, build, and install PyYAML.
RUN mkdir /tmp/pyyaml && \
  cd /tmp/pyyaml && \
  curl -o pyyaml-6.0.1.tar.gz -L https://github.com/yaml/pyyaml/archive/refs/tags/6.0.1.tar.gz && \
  tar xzf pyyaml-6.0.1.tar.gz && \
  cd pyyaml-6.0.1 && \
  python3 setup.py --with-libyaml install

# orjson is also special.  The wheels on PyPI rely on glibc, so we
# need to use cargo/rustc/patchelf to build a musl-compatible version.
RUN pip3 install orjson==3.9.7
