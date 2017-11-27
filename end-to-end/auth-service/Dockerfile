FROM ubuntu:14.04

MAINTAINER Datawire <flynn@datawire.io>
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

# We need curl, pip, and dnsutils (for nslookup).
RUN apt-get update && apt-get -q install -y \
    curl \
    python3-pip \
    dnsutils

# Set WORKDIR to /application which is the root of all our apps then COPY
# only requirements.txt to avoid screwing up Docker caching and causing a
# full reinstall of all dependencies when dependencies are not changed.

WORKDIR /application
COPY requirements.txt .

# Install application dependencies
RUN pip3 install -r requirements.txt

# COPY the app code and configuration into place, then perform any final
# configuration steps.

COPY auth.py authsvc.crt authsvc.key ./

# COPY the entrypoint script and make it runnable.
COPY entrypoint.sh .
RUN chmod 755 entrypoint.sh

ARG TLS=""
ENV TLS=${TLS}

ENTRYPOINT ./entrypoint.sh ${TLS}
