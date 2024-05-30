# The `test-auth` image gets built by `build-aux/check.mk` for use by
# `python/tests/ingegration/manifests.py`.

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

FROM ubuntu:23.10

LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
    PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
    DESCRIPTION              = "Ambassador REST Service" \
    VENDOR                   = "Datawire" \
    VENDOR_URL               = "https://datawire.io/"

# This Dockerfile is set up to install all the application-specific stuff into
# /application.
#
# NOTE: If you don't know what you're doing, it's probably a mistake to
# blindly hack up this file.

# We need curl, pip, and dnsutils (for nslookup) -- and dnsutils mustn't
# be trying to prompt for time zones or the like.
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get -q install -y \
    curl \
    python3-pip \
    python3-venv \
    dnsutils

# Set WORKDIR to /application which is the root of all our apps.
WORKDIR /application

# Create the Python virtual environment
RUN python3 -m venv /opt/venv

# COPY only requirements.txt to avoid screwing up Docker caching and
# causing a full reinstall of all dependencies when dependencies are
# not changed.
COPY docker/test-auth/requirements.txt .
# Install application dependencies
RUN /opt/venv/bin/pip3 install -r requirements.txt

# COPY the app code and configuration into place.
COPY docker/test-auth/auth.py docker/test-auth/authsvc.crt docker/test-auth/authsvc.key docker/test-auth/entrypoint.sh ./

# perform any final configuration steps.
ARG TLS=""
ENV TLS=${TLS}
ENTRYPOINT ./entrypoint.sh ${TLS}
