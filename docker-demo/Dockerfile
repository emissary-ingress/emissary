FROM datawire/qotm:1.1 as qotm

FROM dwflynn/ambassador:0.14.1-DIRTY

WORKDIR /demo

# Copy in the qotm service...
COPY --from=qotm /service/qotm/qotm.py .
#COPY qotm.py .

# ...and the auth service...
COPY simple-auth-server.py .

# ...and the demo config.
COPY config /etc/ambassador-config

# Finally set an endpoint.

# COPY the entrypoint and probes scripts and make them runnable.
COPY entrypoint.sh probes.sh ./
RUN chmod 755 entrypoint.sh probes.sh

ENTRYPOINT [ "./entrypoint.sh" ]
