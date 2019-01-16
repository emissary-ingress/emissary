#!/bin/sh

/usr/sbin/sshd -e
exec /app/proxy
