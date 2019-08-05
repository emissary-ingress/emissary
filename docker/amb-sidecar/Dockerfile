FROM alpine:3.8
RUN apk add --no-cache \
		ca-certificates bash libc6-compat \
	&& addgroup -g 1000 -S datawire \
	&& adduser -u 1000 -S datawire -G datawire

COPY ./amb-sidecar /opt/ambassador-pro/
COPY ./*.opensource.tar.gz /

USER datawire
CMD [ "/opt/ambassador-pro/amb-sidecar" ]
EXPOSE 8500
