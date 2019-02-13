#FROM quay.io/datawire/ambassador_pro:amb-sidecar-0.1.3-plugins1
FROM localhost:31000/amb-sidecar:0.1.3-plugins1

COPY ./*.so /etc/ambassador-plugins/
