#
# To use run from within a checkout of devportal-content repo:
#
#        docker --rm --volume $PWD:/content --publish 8877:8877 <this image>
#
# and visit http://localhost:8877
#
FROM alpine:3.8
#
# The volume /content is expected to be bind-mounted to a checkout of
# https://github.com/datawire/devportal-content.git
#
VOLUME /content
#
# Local devportal will be accessible on this port
#
EXPOSE 8877/tcp
COPY local-devportal /usr/local/ambassador/
ENTRYPOINT ["/usr/local/ambassador/local-devportal", "serve", "/content" ]
