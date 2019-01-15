FROM alpine:3.8
RUN apk add --no-cache ca-certificates

COPY ./ambassador-oauth /bin/
CMD ["/bin/ambassador-oauth"]
EXPOSE 8080
