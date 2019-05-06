FROM alpine:3.9
RUN apk add --no-cache ca-certificates bash libc6-compat

COPY ./run.sh ./dev-portal-server /opt/ambassador-pro/

CMD [ "/opt/ambassador-pro/run.sh" ]
EXPOSE 8680
