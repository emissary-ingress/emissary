ARG envoy="i-forgot-to-set-build-arg-envoy"
ARG base_python="i-forgot-to-set-build-arg-base_python"

FROM ${envoy} as envoy

FROM ${base_python}

# Always have an "ambassador" user as UID 8888. This is what we recommend
# people run as. (Note that the "-D" actually leaves the password locked.)
RUN adduser ambassador -u 8888 -G root -D -H -s /bin/false

RUN ln -s /usr/bin/python3 /usr/bin/python

# Other installers
COPY --from=artifacts /opt/image-build /opt/image-build
RUN /opt/image-build/install.sh

# Our envoy. The capabilities here grant the wrapper the ability to use the
# cap_net_bind_service cap and for Envoy to inherit it.
COPY --from=envoy /usr/local/bin/envoy-static-stripped /usr/local/bin/envoy
RUN setcap cap_net_bind_service=ei /usr/local/bin/envoy

# Our Go binaries. See envoy section for setcap info.
COPY --from=artifacts /opt/ambassador /opt/ambassador
RUN ln -s /opt/ambassador/bin/* /usr/local/bin/
RUN setcap cap_net_bind_service=p /opt/ambassador/bin/wrapper

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

# Force the HOME environment variable to a directory that'll always be writeable.
# We use /tmp/ambassador for this, and make sure it exists in our entrypoint,
# because trying to create it here in the Dockerfile doesn't always work very
# well in the face of situations like KAT volume-mounting /tmp/ambassador or
# the like.
ENV HOME=/tmp/ambassador

COPY --from=artifacts /bin/busybox /bin/busybox

ENTRYPOINT [ "bash", "/buildroot/ambassador/python/entrypoint.sh" ]
