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
# This Dockerfile copies in the required packages from the BASE_PY_IMAGE and configures
# Ambassador. The reason packages are not installed and copied instead is that this pattern speeds up
# the inner development loop, since packages do not need to be installed in every docker build.
# Besides that, alpine mirrors have been found to be inconsistent over timezones, so this seems to
# be a better approach.

########
# Seeing weird errors copying stuff? Check .dockerignore!!
########

# By default, Ambassador's config and other application-specific stuff gets written to /ambassador. You can
# configure a different location for the runtime configuration elements via environment variables.

# Arguments ####################################################################
ARG BASE_RUNTIME_IMAGE
ARG BASE_PY_IMAGE
FROM $BASE_RUNTIME_IMAGE as base-runtime
FROM $BASE_PY_IMAGE as base-py

# Image: staging-envoy #########################################################
FROM base-runtime as staging-envoy
# ADD/COPY the file in, then reset its timestamp to the unix epoch, so
# the timestamp doesn't break Docker layer caching.
ARG ENVOY_FILE
ADD $ENVOY_FILE /usr/local/bin/envoy
RUN touch -t 197001010000 /usr/local/bin/envoy

# Image: stage1 ################################################################
# STAGE ONE: use the BASE_PY_IMAGE's toolchains to
# build and install the Ambassador app itself.
FROM base-py as stage1
# Install the application itself
COPY python/ ambassador
RUN rm -rf ./multi
RUN cd ambassador && python3 setup.py --quiet install
RUN rm -rf ./ambassador

# Image: (final) ###############################################################
# STAGE TWO: switch to the BASE_GO_IMAGE as the base of
# our actual runtime image, and copy the built artifacts from
# stage one to here.
FROM base-runtime

MAINTAINER Datawire <flynn@datawire.io>
LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
      PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
      DESCRIPTION              = "Ambassador" \
      VENDOR                   = "Datawire" \
      VENDOR_URL               = "https://datawire.io/"

ENV AMBASSADOR_ROOT=/ambassador
WORKDIR ${AMBASSADOR_ROOT}

COPY --from=staging-envoy /usr/local/bin/envoy /usr/local/bin/envoy

# One could argue that this is perhaps a bit of a hack. However, it's also the way to
# get all the stuff that pip installed without needing the whole of the Python dev
# chain.
COPY --from=stage1 /usr/lib/python3.7 /usr/lib/python3.7/
COPY --from=stage1 /usr/lib/libyaml* /usr/lib/
COPY --from=stage1 /usr/lib/pkgconfig /usr/lib/

# Copy Ambassador binaries (built in stage one).
COPY --from=stage1 /usr/bin/ambassador /usr/bin/diagd /usr/bin/

# MKDIR an empty /ambassador/ambassador-config, so that you can drop a configmap over it
# if you really really need to (not recommended).
RUN mkdir ambassador-config envoy

# COPY in the stuff for use with --demo.
COPY demo/config/ ambassador-demo-config
COPY demo/services/ demo-services

# ...and symlink /usr/bin/python because of !*@&#*!@&# Werkzeug.
RUN ln -s python3 /usr/bin/python

# Fix permissions to allow running as a non root user
RUN chgrp -R 0 ${AMBASSADOR_ROOT} && \
    chmod -R u+x ${AMBASSADOR_ROOT} && \
    chmod -R g=u ${AMBASSADOR_ROOT} /etc/passwd

# COPY the entrypoint and Python-kubewatch and make them runnable.
COPY python/entrypoint.sh .
COPY python/grab-snapshots.py .
COPY python/kick_ads.sh .
COPY python/kubewatch.py .
COPY python/post_update.py .
COPY python/watch_hook.py .
RUN chmod 755 entrypoint.sh grab-snapshots.py kick_ads.sh kubewatch.py post_update.py watch_hook.py

COPY bin_linux_amd64/ambex bin_linux_amd64/kubestatus bin_linux_amd64/watt bin_linux_amd64/kubectl /usr/local/bin/

RUN setcap 'cap_net_bind_service=+ep' /usr/local/bin/envoy
ENTRYPOINT [ "./entrypoint.sh" ]
