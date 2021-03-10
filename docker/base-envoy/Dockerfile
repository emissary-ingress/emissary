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
# If you have to change this file, you _must_ update BASE_ENVOY_RELVER
# in the Makefile, then run "make update-base" to build and push the
# new image.

ARG base="i-forgot-to-set-build-arg-base"
FROM ${base}

ADD ./envoy-static ./envoy-static-stripped /usr/local/bin/
