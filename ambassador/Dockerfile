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

FROM envoyproxy/envoy-alpine:26e2a31650fe14db7e55a3136f9ae75db667bf5c

MAINTAINER Datawire <flynn@datawire.io>
LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
      PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
      DESCRIPTION              = "Ambassador" \
      VENDOR                   = "Datawire" \
      VENDOR_URL               = "https://datawire.io/"

# This Dockerfile is set up to install all the application-specific stuff into
# /ambassador.
#
# NOTE: If you don't know what you're doing, it's probably a mistake to
# blindly hack up this file.

RUN apk --no-cache add curl python3 socat

# Set WORKDIR to /ambassador which is the root of all our apps then COPY
# only requirements.txt to avoid screwing up Docker caching and causing a
# full reinstall of all dependencies when dependencies are not changed.
ENV AMBASSADOR_ROOT=/ambassador
WORKDIR ${AMBASSADOR_ROOT}
COPY requirements.txt .

# Install application dependencies
RUN pip3 install -r requirements.txt

# Install the application itself
COPY ./ ambassador
RUN cd ambassador && python3 setup.py --quiet install
RUN rm -rf ./ambassador

# MKDIR an empty /ambassador/ambassador-config. You can dump a
# configmap over this with no trouble, or you can let
# annotations do the right thing
RUN mkdir ambassador-config
RUN mkdir envoy

# COPY in a default config for use with --demo.
COPY default-config/ ambassador-demo-config

# Fix permissions to allow running as a non root user
RUN chgrp -R 0 ${AMBASSADOR_ROOT} && \
    chmod -R u+x ${AMBASSADOR_ROOT} && \
    chmod -R g=u ${AMBASSADOR_ROOT} /etc/passwd

# COPY the entrypoint script and make it runnable.
COPY kubewatch.py .
COPY entrypoint.sh .
# XXX: ambex needs to be replaced with a proper release, or made traceable or something!!!
COPY ambex .
COPY bootstrap-ads.yaml .
RUN chmod 755 entrypoint.sh

ENTRYPOINT [ "./entrypoint.sh" ]
