FROM quay.io/datawire/ambassador_pro:amb-sidecar-0.2.0-rc1

COPY ./*.so /etc/ambassador-plugins/
