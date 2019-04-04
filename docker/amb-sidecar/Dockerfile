FROM alpine:3.8
RUN apk add --no-cache \
		ca-certificates bash libc6-compat \
	&& addgroup -g 1000 -S datawire \
	&& adduser -u 1000 -S datawire -G datawire

COPY ./run.sh ./amb-sidecar /opt/ambassador-pro/

USER datawire
CMD [ "/opt/ambassador-pro/run.sh" ]
EXPOSE 6070 8081 8082
