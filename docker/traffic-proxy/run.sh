#!/bin/sh

/usr/sbin/sshd -e
exec /app/traffic-proxy
