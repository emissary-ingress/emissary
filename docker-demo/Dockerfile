FROM datawire/qotm:1.3 as qotm

FROM ambassador:0.14.2-b5.0a53228.DIRTY

WORKDIR /demo

# Copy in the qotm service...
COPY --from=qotm /service/qotm/qotm.py .

COPY check-services.py .

# ...and the auth service...
COPY simple-auth-server.py .

# ...and the prober...
COPY probes.sh .

# ...and the demo config.
COPY config /etc/ambassador-config

# Finally set an endpoint.

# COPY the entrypoint script and make it runnable.
COPY entrypoint.sh .
RUN chmod 755 entrypoint.sh

ENTRYPOINT [ "./entrypoint.sh" ]
