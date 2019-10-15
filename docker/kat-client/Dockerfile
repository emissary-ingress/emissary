ARG BASE_PY_IMAGE
FROM $BASE_PY_IMAGE

RUN apk add iptables curl bind-tools

WORKDIR /work

COPY teleproxy kat_client ./
COPY entrypoint.sh ./entrypoint.sh

RUN chmod 755 teleproxy kat_client entrypoint.sh

ENTRYPOINT [ "./entrypoint.sh" ]

