FROM alpine:3.9
RUN apk add --no-cache ca-certificates bash libc6-compat

COPY ./run.sh ./apro-internal-access /opt/ambassador-pro/

CMD [ "/opt/ambassador-pro/run.sh" ]
EXPOSE 5000
