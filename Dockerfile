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
# This is a multistage Dockerfile, because while we need the compilers et al for the build, we don't need them
# to run things. If you don't know what you're doing, it's probably a mistake to blindly hack up this file.
#
# By default, Ambassador's config and other application-specific stuff gets written to /ambassador. You can
# configure a different location for the runtime configuration elements via environment variables.
#
# Ambassador itself is installed with pip, so it ends up in /usr/bin and /usr/lib. At present, other
# executables are written into /ambassador, but we'll likely change that shortly.
#
# The base image (for everything) is defined here.

FROM quay.io/datawire/ambassador-envoy-alpine-stripped:v1.8.0-g14e2c65bb as BASE

MAINTAINER Datawire <flynn@datawire.io>
LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
      PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
      DESCRIPTION              = "Ambassador" \
      VENDOR                   = "Datawire" \
      VENDOR_URL               = "https://datawire.io/"

################################################################
## FIRST STAGE: this is where we do compiles and pip installs and all that.

FROM BASE as builder
ENV AMBASSADOR_ROOT=/ambassador

# Compilers and pip and all that good stuff go here.
RUN apk --no-cache add go build-base libffi-dev openssl-dev python3-dev
RUN pip3 install -U pip

# Set WORKDIR to /ambassador which is the root of all our apps then COPY
# only requirements.txt to avoid screwing up Docker caching and causing a
# full reinstall of all dependencies when dependencies are not changed.

WORKDIR ${AMBASSADOR_ROOT}

COPY releng releng
COPY multi/requirements.txt multi/
COPY ambassador/requirements.txt ambassador/

# Install application dependencies
RUN releng/install-py.sh prd requirements */requirements.txt

# Install the application itself
COPY multi/ multi
COPY ambassador/ ambassador
RUN releng/install-py.sh prd install */requirements.txt
RUN rm -rf ./multi ./ambassador

# Grab kubewatch
RUN wget -q https://s3.amazonaws.com/datawire-static-files/kubewatch/0.3.13/$(go env GOOS)/$(go env GOARCH)/kubewatch
RUN chmod +x kubewatch

# Clean up no-longer-needed dev stuff.
# RUN apk del build-base libffi-dev openssl-dev python3-dev go

################################################################
## SECOND STAGE: this is where we pull over the stuff we need to actually run Ambassador,
## _without_ all the compilers and crap.

FROM BASE as foundation
ENV AMBASSADOR_ROOT=/ambassador
WORKDIR ${AMBASSADOR_ROOT}

RUN echo "https://mirror.math.princeton.edu/pub/alpinelinux/v3.8/main" > /etc/apk/repositories && \
    echo "https://mirror.math.princeton.edu/pub/alpinelinux/v3.8/community" >> /etc/apk/repositories

RUN apk --no-cache add curl python3

# One could argue that this is perhaps a bit of a hack. However, it's also the way to
# get all the stuff that pip installed without needing the whole of the Python dev
# chain.
COPY --from=builder /usr/lib/python3.6 /usr/lib/python3.6

# Copy Ambassador binaries...
COPY --from=builder /usr/bin/ambassador /usr/bin/diagd /usr/bin/

# ...and go-kubewatch.
COPY --from=builder ${AMBASSADOR_ROOT}/kubewatch ${AMBASSADOR_ROOT}/

# MKDIR an empty /ambassador/ambassador-config, so that you can drop a configmap over it
# if you really really need to (not recommended).
RUN mkdir ambassador-config
RUN mkdir envoy

# COPY in a default config for use with --demo.
COPY ambassador/default-config/ ambassador-demo-config

# Fix permissions to allow running as a non root user
RUN chgrp -R 0 ${AMBASSADOR_ROOT} && \
    chmod -R u+x ${AMBASSADOR_ROOT} && \
    chmod -R g=u ${AMBASSADOR_ROOT} /etc/passwd

# COPY the entrypoint and Python-kubewatch and make them runnable.
COPY ambassador/kubewatch.py .
COPY ambassador/entrypoint.sh .
RUN chmod 755 kubewatch.py entrypoint.sh

# Grab ambex, too.
RUN wget -q https://s3.amazonaws.com/datawire-static-files/ambex/0.1.1/ambex
RUN chmod 755 ambex

ENTRYPOINT [ "./entrypoint.sh" ]
