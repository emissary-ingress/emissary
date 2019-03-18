# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

################################################################
# This image has the toolchains we need to actually build Ambassador, plus
# the Python dependencies that Ambassador actually needs. It's used as a
# base image in the main Dockerfile, so that we can use the Docker cache
# to more effectively dodge network issues with APK repos or the Cheese
# Shop.
#
# If you have to change this - including changing Ambassador's dependency
# graph, including multi! - you _must_ update AMBASSADOR_DOCKER_IMAGE_CACHED
# in the Makefile, then run "make docker-update-base" to build and push the
# new image.

ARG ENVOY_BASE_IMAGE
FROM $ENVOY_BASE_IMAGE

MAINTAINER Datawire <flynn@datawire.io>

LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
      PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
      DESCRIPTION              = "Ambassador" \
      VENDOR                   = "Datawire" \
      VENDOR_URL               = "https://datawire.io/"

ENV AMBASSADOR_ROOT=/ambassador
WORKDIR ${AMBASSADOR_ROOT}

RUN apk --no-cache add go build-base libffi-dev openssl-dev python3-dev python3 curl && \
    pip3 install -U pip

# The YAML parser is... special. First, we need to install libyaml. No,
# I don't know why it's called yaml-dev.
RUN apk --no-cache add yaml-dev

# Next, we need Cython. Sadly, we can't `apk add cython3`, so we have
# to build the silly thing.
RUN pip3 install cython

# After that, we can download !*@&#!*@&# PyYAML...
RUN curl -O -L http://pyyaml.org/download/pyyaml/PyYAML-3.13.tar.gz
RUN tar xzf PyYAML-3.13.tar.gz

# ...and then install it.
RUN cd PyYAML-3.13 && python3 setup.py --with-libyaml install

COPY releng releng
COPY multi/requirements.txt multi/
COPY ambassador/requirements.txt ambassador/

# Install application dependencies
RUN releng/install-py.sh prd requirements */requirements.txt
