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
# the binaries and entrypoints for any image that we produce, and
# therefore (with the appropriate entrypoint) can function in place of
# any release image. The release images are trivial stages that simply
# copy the relevant artifacts out of a snapshot of the builder
# container; these release images are marked with the magic comment
# "#external".

# https://www.practicemp.com/2018/10/docker-dockerfile-reference.html

# This argument controlls the base image is used for both our build
# container and for the release containers.
# 这个镜像需要自行构建出支持 ARM64 架构的版本
# https://hub.docker.com/repository/docker/zjingjie/alpine-glibc/tags?name=alpine-3.10
# ARG base="frolvlad/alpine-glibc:alpine-3.10"
ARG base="zjingjie/alpine-glibc:alpine-3.10-arm64"
 
# This argument controls where the release images get their artifacts
# from. We override it for incremental builds, but default it to the
# builder stage so this Dockerfile works out of the box.
# 这个是本地构建的镜像，包含了大量依赖文件，可以试着不通过构建，自行编译直接复制进来
ARG artifacts="builder"
 
# This controls where we copy envoy from.
# 这个镜像需要自行构建出支持 ARM64 架构的版本
# Ambassador v0.86.1 使用的是 v1.12.1 版本，但其依赖不支持构建出 ARM64 版本
# 能够在 ARM64 架构的机器上构建出来的最近版本是：v1.15.0
# https://hub.docker.com/repository/docker/zjingjie/envoy/tags?name=v1.15.0
# ARG envoy="quay.io/datawire/ambassador-base:envoy-7.d17d947caef13f1bdd235c3fccff77814883bb46.opt"
ARG envoy="zjingjie/envoy:v1.15.0-arm64"
 
########################################
# The builder image
########################################
 
FROM ${envoy} as envoy
 
FROM ${base} as builder
 
WORKDIR /buildroot
 
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/buildroot/bin
 
# The YAML parser is... special. To get the C version, we need to install Cython and libyaml, then
# build it locally -- just using pip won't work.
 
# 这里要替换掉原有的 AMD64 架构包
# https://dl.google.com/go/go1.13.linux-amd64.tar.gz 替换为 https://golang.org/dl/go1.13.linux-arm64.tar.gz
# https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl 替换为 https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/arm64/kubectl
RUN apk --no-cache add bash curl jq rsync python3 python3-dev build-base libffi-dev openssl-dev sudo \
                       iptables docker openssh-client libcap libcap-dev yaml-dev cython git && \
    pip3 install -U pip && \
    curl --fail https://golang.org/dl/go1.13.linux-arm64.tar.gz | tar -C /usr/local -xzf - && \
    curl --fail -L https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/arm64/kubectl -o /usr/bin/kubectl && \
    chmod a+x /usr/bin/kubectl && \
    ln -s /usr/bin/python3 /usr/bin/python
 
RUN chmod u+s $(which docker)
 
COPY --from=envoy /usr/local/bin/envoy /usr/local/bin/envoy
 
ENV KUBECONFIG=/buildroot/kubeconfig.yaml
 
# Download, build, and install PyYAML.
RUN mkdir /tmp/pyyaml && \
    cd /tmp/pyyaml && \
    curl -O -L http://pyyaml.org/download/pyyaml/PyYAML-5.1.tar.gz && \
    tar xzf PyYAML-5.1.tar.gz && \
    cd PyYAML-5.1 && \
    python3 setup.py --with-libyaml install

# Then we can do the rest of the Python stuff.
# requirements 文件存储路径：./builder/requirements.txt
COPY requirements.txt .
RUN pip install -r requirements.txt
 
# XXX: this will go away
# watt 这个应该是需要用 go 自行编译
# py 脚本存储路径：./python/
RUN mkdir /ambassador && \
    ln -s /buildroot/bin/watt /ambassador/watt && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py
 
RUN adduser dw --disabled-password
# SUDO_USERS HOSTS=(AS_USER) TAGS COMMANDS
RUN echo "dw ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/datawire
RUN chown dw /buildroot
# 设置用户名，以供在运行镜像时使用，以及在 Dockerfile 中声明 USER 指令后的运行任何 RUN，CMD 和 ENTRYPOINT 指令时使用
USER dw
 
ENTRYPOINT [ "/bin/bash" ]
 
 
########################################
# The optimized images
########################################
 
FROM ${artifacts} as artifacts
 
#external
FROM ${base} as ambassador
 
RUN apk --no-cache add bash curl python3 libcap
 
RUN ln -s /usr/bin/python3 /usr/bin/python
 
COPY --from=builder /usr/local/bin/envoy /usr/local/bin/envoy
COPY --from=builder /usr/lib/python3.7/site-packages /usr/lib/python3.7/site-packages
COPY --from=builder /usr/lib/libyaml* /usr/lib/
 
COPY --from=artifacts /buildroot/bin/ambex /usr/bin/ambex
COPY --from=artifacts /buildroot/bin/watt /usr/bin/watt
COPY --from=artifacts /buildroot/bin/kubestatus /usr/bin/kubestatus
COPY --from=artifacts /usr/bin/kubectl /usr/bin/kubectl
COPY --from=artifacts /buildroot/ambassador/python /buildroot/ambassador/python
COPY --from=artifacts /buildroot/bin/capabilities_wrapper /usr/local/bin/wrapper
RUN cd /buildroot/ambassador/python && python setup.py install
 
# XXX: this will go away
RUN mkdir /ambassador && \
    ln -s /usr/bin/watt /ambassador/watt && \
    ln -s /usr/bin/kubestatus /ambassador/kubestatus && \
    ln -s /buildroot/ambassador/python/post_update.py /ambassador/post_update.py && \
    ln -s /buildroot/ambassador/python/watch_hook.py /ambassador/watch_hook.py && \
    ln -s /buildroot/ambassador/python/kubewatch.py /ambassador/kubewatch.py
 
COPY --from=artifacts /buildroot/ambassador/demo/config /ambassador/ambassador-demo-config
COPY --from=artifacts /buildroot/ambassador/demo/services /ambassador/demo-services
 
WORKDIR /ambassador
 
# Fix permissions to allow correctly running as a non root user
RUN chgrp -R 0 /ambassador && \
    chmod -R u+x /ambassador && \
    chmod -R g=u /ambassador /etc/passwd
# The capabilities here grant the wrapper the ability to use the cap_net_bind_service cap
# and for Envoy to inherit it.
RUN setcap cap_net_bind_service=p /usr/local/bin/wrapper; setcap cap_net_bind_service=ei /usr/local/bin/envoy
 
ENTRYPOINT [ "bash", "/buildroot/ambassador/python/entrypoint.sh" ]
 
########################################
 
#external
FROM ${base} as kat-client
 
COPY --from=artifacts /buildroot/bin/kat-client /usr/bin/kat-client
RUN mkdir /work && ln -s /usr/bin/kat-client /work/kat_client
CMD [ "sleep", "3600" ]
 
########################################
 
#external
FROM ${base} as kat-server
 
COPY --from=artifacts /buildroot/bin/kat-server /usr/bin/kat-server
 
WORKDIR /work
COPY server.crt server.crt
COPY server.key server.key
 
ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api
 
EXPOSE 8080
CMD [ "kat-server" ]
