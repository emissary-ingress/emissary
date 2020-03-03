###
# This dockerfile builds all the source code and docker images for the
# edge stack.
##

###
# How it works.
##
# All our source code is built in a single monolithic build
# container. (The "builder" stage below.) We then produce optimized
# release images by copying the specific binaries and entrypoints out
# of the build container. The build container must always contain all
# the binaries and entrypoints for any container we produce, and
# therefore (with the appropriate entrypoint) can function in place of
# any release container. The release containers are trivial stages
# that simply copy the relevant artifacts out of the builder
# container.

# This argument controlls the base image is used for both our build
# container and for the release containers.
ARG base="frolvlad/alpine-glibc:alpine-3.10"

# This argument controls where the release images get their artifacts
# from. We override it for incremental builds, but default it to the
# builder stage so this Dockerfile works out of the box.
ARG artifacts="builder"

# This controls where we copy envoy from.
ARG envoy="i-forgot-to-set-build-arg-envoy"

########################################
# The builder image
########################################

FROM ${envoy} as envoy

FROM ${base} as builder

WORKDIR /buildroot

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin

# The YAML parser is... special. To get the C version, we need to install Cython and libyaml, then
# build it locally -- just using pip won't work.

RUN apk --no-cache add bash bash-completion ncurses curl jq rsync python3 python3-dev build-base libffi-dev openssl-dev sudo \
                       iptables docker openssh-client libcap libcap-dev yaml-dev cython git nodejs npm && \
    pip3 install -U pip && \
    curl --fail https://dl.google.com/go/go1.13.linux-amd64.tar.gz | tar -C /usr/local -xzf - && \
    curl --fail -L https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl -o /usr/bin/kubectl && \
    chmod a+x /usr/bin/kubectl && \
    curl --fail -L https://raw.githubusercontent.com/ahmetb/kubectx/v0.7.0/kubectx -o /usr/local/bin/kubectx && \
    curl --fail -L https://raw.githubusercontent.com/ahmetb/kubectx/v0.7.0/kubens -o /usr/local/bin/kubens && \
    chmod a+x /usr/local/bin/kubectx /usr/local/bin/kubens && \
    curl --fail -L https://raw.githubusercontent.com/ahmetb/kubectx/v0.7.0/completion/kubectx.bash -o /usr/share/bash-completion/completions/kubectx && \
    curl --fail -L https://raw.githubusercontent.com/ahmetb/kubectx/v0.7.0/completion/kubens.bash -o /usr/share/bash-completion/completions/kubens && \
    curl -L https://raw.githubusercontent.com/jonmosco/kube-ps1/v0.7.0/kube-ps1.sh -o /usr/local/bin/kube-ps1.sh && \
    ln -s /usr/bin/python3 /usr/bin/python && \
    npm config set user 0 && \
    npm config set unsafe-perm true && \
    npm install -g rollup rollup-plugin-terser && \
    curl https://sdk.cloud.google.com > /tmp/install.sh && \
    mkdir /usr/lib64/ && \
    bash /tmp/install.sh --disable-prompts --install-dir=/usr/lib64

RUN sh -c "/usr/bin/kubectl completion bash > /usr/share/bash-completion/completions/kubectl"
RUN chmod u+s $(which docker)

COPY --from=envoy /usr/local/bin/envoy-static-stripped /usr/local/bin/envoy

ENV KUBECONFIG=/buildroot/kubeconfig.yaml

# Download, build, and install PyYAML.
RUN mkdir /tmp/pyyaml && \
    cd /tmp/pyyaml && \
    curl -O -L http://pyyaml.org/download/pyyaml/PyYAML-5.1.tar.gz && \
    tar xzf PyYAML-5.1.tar.gz && \
    cd PyYAML-5.1 && \
    python3 setup.py --with-libyaml install

# Then we can do the rest of the Python stuff.
COPY requirements.txt .
RUN pip install -r requirements.txt

# XXX: this will go away
RUN mkdir -p /ambassador/sidecars && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py

RUN adduser dw --disabled-password
# SUDO_USERS HOSTS=(AS_USER) TAGS COMMANDS
RUN echo "dw ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/datawire
RUN chown dw /buildroot
USER dw

ENTRYPOINT [ "/bin/bash" ]


########################################
# The optimized images
########################################

FROM ${artifacts} as artifacts

FROM ${base} as ambassador

# External stuff that should change infrequently
RUN apk --no-cache add bash curl python3 libcap
RUN ln -s /usr/bin/python3 /usr/bin/python
COPY --from=artifacts /usr/bin/kubectl /usr/bin/kubectl
COPY --from=artifacts /usr/lib/libyaml* /usr/lib/

# External Python packages we use
COPY --from=artifacts /usr/lib/python3.7/site-packages /usr/lib/python3.7/site-packages

# Our envoy. The capabilities here grant the wrapper the ability to use the
# cap_net_bind_service cap and for Envoy to inherit it.
COPY --from=envoy /usr/local/bin/envoy-static-stripped /usr/local/bin/envoy
RUN setcap cap_net_bind_service=ei /usr/local/bin/envoy

# Our Go binaries. See envoy section for setcap info.
COPY --from=artifacts /opt/ambassador /opt/ambassador
RUN ln -s /opt/ambassador/bin/* /usr/local/bin/
RUN setcap cap_net_bind_service=p /usr/local/bin/wrapper

# Our Python code
COPY --from=artifacts /buildroot/ambassador/python /buildroot/ambassador/python
RUN cd /buildroot/ambassador/python && python setup.py install

# Configuration, Docker demo stuff, the AES WebUI. The /ambassador bit changes
# in post-install so it's always stale. But it's pretty small, so it's not too
# bad to re-push every time.
COPY --from=artifacts /ambassador /ambassador
COPY --from=artifacts /buildroot/ambassador/demo/config /ambassador/ambassador-demo-config
COPY --from=artifacts /buildroot/ambassador/demo/services /ambassador/demo-services

# Fix permissions to allow correctly running as a non root user
# XXX: We could combine everything into one tree in the builder, fix permissions
# there, and then a use single COPY to get everything and avoid duplicating the
# (small amount of) data in a new layer for this RUN.
RUN chgrp -R 0 /ambassador && \
    chmod -R u+x /ambassador && \
    chmod -R g=u /ambassador /etc/passwd

WORKDIR /ambassador

ENTRYPOINT [ "bash", "/buildroot/ambassador/python/entrypoint.sh" ]

########################################

FROM ${base} as kat-client

COPY --from=artifacts /buildroot/bin/kat-client /usr/bin/kat-client
RUN mkdir /work && ln -s /usr/bin/kat-client /work/kat_client
CMD [ "sleep", "3600" ]

########################################

FROM ${base} as kat-server

COPY --from=artifacts /buildroot/bin/kat-server /usr/bin/kat-server

WORKDIR /work
COPY server.crt server.crt
COPY server.key server.key

ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api

EXPOSE 8080
CMD [ "kat-server" ]
