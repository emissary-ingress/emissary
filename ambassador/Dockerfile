# Our base image was built from https://github.com/datawire/envoy, commit
# 6557e9ea7, on the flynn/feature/extauth branch. We need this instead of
# the official Envoy image so that we can support Ambassador's ability to
# use an external authentication service.
#
# To reproduce the base image,
#
# git clone https://github.com/datawire/envoy.git datawire-envoy
# cd datawire-envoy
# git checkout flynn/feature/extauth
# 
# and then read DATAWIRE/README.md.

# FROM datawire/ambassador-envoy-alpine-stripped:v1.5.0-232-g6557e9ea7
FROM datawire/ambassador-envoy-alpine-stripped:v1.6.0-343-g0e38b716

MAINTAINER Datawire <flynn@datawire.io>
LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
      PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
      DESCRIPTION              = "Ambassador" \
      VENDOR                   = "Datawire" \
      VENDOR_URL               = "https://datawire.io/"

# This Dockerfile is set up to install all the application-specific stuff into
# /application.
#
# NOTE: If you don't know what you're doing, it's probably a mistake to
# blindly hack up this file.

RUN apk --no-cache add curl python3

# Set WORKDIR to /application which is the root of all our apps then COPY
# only requirements.txt to avoid screwing up Docker caching and causing a
# full reinstall of all dependencies when dependencies are not changed.

WORKDIR /application
COPY requirements.txt .

# Install application dependencies
RUN pip3 install -r requirements.txt

# Install the application itself
COPY ./ ambassador
RUN cd ambassador && python3 setup.py --quiet install
RUN rm -rf ./ambassador

# MKDIR an empty /etc/ambassador-config. You can dump a configmap over this with no
# trouble, or you can let annotations do the right thing.
RUN mkdir /etc/ambassador-config

# COPY in a default config for use with --demo.
COPY default-config/ /etc/ambassador-demo-config

# COPY the entrypoint script and make it runnable.
COPY kubewatch.py .
COPY hot-restarter.py .
COPY start-envoy.sh .
COPY entrypoint.sh .
RUN chmod 755 start-envoy.sh entrypoint.sh

ENTRYPOINT [ "./entrypoint.sh" ]
