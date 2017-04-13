#!/bin/sh

# set -x

CHAINPATH="$1"
PRIVKEYPATH="$2"
PREFIX="$3"
SECRETNAME="${4:-ambassador-certs}"
NAMESPACE="${5:-default}"

if [ -z "$CHAINPATH" -o -z "PRIVKEYPATH" ]; then
    echo "Usage: $(basename $0) chainpath privkeypath [prefix [secretname [namespace]]]" >&2
    exit 1
fi

errors=

if [ ! -r "$CHAINPATH" ]; then
    echo "$CHAINPATH is not readable" >&2
    errors=Y
fi

if [ ! -r "$PRIVKEYPATH" ]; then
    echo "$PRIVKEYPATH is not readable" >&2
    errors=Y
fi

if [ -n "$errors" ]; then
    exit 1
fi

CHAINNAME="fullchain.pem"
PRIVKEYNAME="privkey.pem"

if [ -n "$PREFIX" ]; then
    CHAINNAME="${PREFIX}-fullchain.pem"
    PRIVKEYNAME="${PREFIX}-privkey.pem"
fi

(cat << EOF
apiVersion: v1
kind: Secret
metadata:
  name: "$SECRETNAME"
  namespace: "$NAMESPACE"
type: Opaque
data:
  "$CHAINNAME": "$(cat "$CHAINPATH" | base64)"
  "$PRIVKEYNAME": "$(cat "$PRIVKEYPATH" | base64)"
EOF
) > secret.yml
kubectl apply -f "secret.yml"
