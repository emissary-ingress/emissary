FROM alpine:3.8
RUN apk add --no-cache ca-certificates bash libc6-compat

COPY ./run.sh ./amb-sidecar /opt/ambassador-pro/

CMD [ "/opt/ambassador-pro/run.sh" ]
EXPOSE 6070 8081 8082
