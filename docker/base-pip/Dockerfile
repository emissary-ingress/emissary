# syntax = docker/dockerfile:1.3

###
# This dockerfile builds the base image for the builder container. See
# the main Dockerfile for more information about what the builder
# container is and how code in this repo is built.
#
# Originally this base was built as part of the builder container's
# bootstrap process. We discovered that minor network interruptions
# would break these steps, and such interruptions were common on our
# cloud CI system. We decided to separate out these steps so that any
# one of them is much less likely to be the cause of a network-related
# failure, i.e. a flake.
#
# See the comment before the build_builder_base() function in builder.sh
# to see when and how often this base image is built and pushed.
##

ARG from="i-forgot-to-set-build-arg-from"

FROM ${from}

COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip  \
  pip3 install -r requirements.txt --ignore-installed packaging \
  && rm requirements.txt
