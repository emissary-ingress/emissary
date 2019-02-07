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

# This Dockerfile copies in the required packages from the CACHED_CONTAINER_IMAGE and configures
# Ambassador. The reason packages are not installed and copied instead is that this pattern speeds up
# the inner development loop, since packages do not need to be installed in every docker build.
# Besides that, alpine mirrors have been found to be inconsistent over timezones, so this seems to
# be a better approach.

# By default, Ambassador's config and other application-specific stuff gets written to /ambassador. You can
# configure a different location for the runtime configuration elements via environment variables.

ARG CACHED_CONTAINER_IMAGE
ARG AMBASSADOR_BASE_IMAGE

# Stage 1
# This copies in the ambassador code to CACHED_CONTAINER_IMAGE and installs it
FROM $CACHED_CONTAINER_IMAGE as cached

# Install the application itself
COPY multi/ multi
COPY ambassador/ ambassador
RUN releng/install-py.sh prd install */requirements.txt
RUN rm -rf ./multi ./ambassador

# Stage 2
# Required dependencies and binaries are copied in from CACHED_CONTAINER_IMAGE to AMBASSADOR_BASE_IMAGE
FROM $AMBASSADOR_BASE_IMAGE
ENV AMBASSADOR_ROOT=/ambassador
WORKDIR ${AMBASSADOR_ROOT}

# One could argue that this is perhaps a bit of a hack. However, it's also the way to
# get all the stuff that pip installed without needing the whole of the Python dev
# chain.
COPY --from=cached /usr/lib/python3.6 /usr/lib/python3.6

# Copy Ambassador binaries...
COPY --from=cached /usr/bin/ambassador /usr/bin/diagd /usr/bin/

# MKDIR an empty /ambassador/ambassador-config, so that you can drop a configmap over it
# if you really really need to (not recommended).
RUN mkdir ambassador-config envoy

# COPY in a default config for use with --demo.
COPY ambassador/default-config/ ambassador-demo-config

# Fix permissions to allow running as a non root user
RUN chgrp -R 0 ${AMBASSADOR_ROOT} && \
    chmod -R u+x ${AMBASSADOR_ROOT} && \
    chmod -R g=u ${AMBASSADOR_ROOT} /etc/passwd

# COPY the entrypoint and Python-kubewatch and make them runnable.
COPY ambassador/kubewatch.py .
COPY ambassador/entrypoint.sh .
COPY ambassador/post_update.py .
RUN chmod 755 kubewatch.py entrypoint.sh post_update.py

ENTRYPOINT [ "./entrypoint.sh" ]
