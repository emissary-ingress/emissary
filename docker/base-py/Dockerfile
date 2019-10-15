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

# If you have to change this file or any of the files that it COPYs
# in, you _must_ update BASE_PY_RELVER in the Makefile, then run "make
# update-base" to build and push the new image.

# Arguments ####################################################################
ARG BASE_RUNTIME_IMAGE
FROM $BASE_RUNTIME_IMAGE as base-runtime

# Image: (final) ###############################################################
# This image caches all of the Python libraries, but is is too big to
# be the base _runtime_ image for Ambassador, because it has:
# compilers, header=files, and other tools.  header-files, and other
# tools.
FROM base-runtime

RUN apk --no-cache add \
	build-base \
	cython3 \
	libffi-dev \
	openssl-dev \
	python3-dev \
	yaml-dev

RUN mkdir /tmp/pyyaml && \
	cd /tmp/pyyaml && \
	curl --fail -O -L http://pyyaml.org/download/pyyaml/PyYAML-3.13.tar.gz && \
	tar xzf PyYAML-3.13.tar.gz && \
	cd PyYAML-3.13 && \
	python3 setup.py --with-libyaml install && \
	cd / && \
	rm -rf /tmp/pyyaml

COPY python/requirements.txt /ambassador/requirements.txt
RUN pip3 install -r /ambassador/requirements.txt
