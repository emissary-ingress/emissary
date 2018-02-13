#!/bin/sh

if [ -n "$HERE" ]; then
    HERE=$(pwd)
fi

KUBERNAUT="$HERE/kubernaut"

get_kubernaut () {
    if [ ! -x "$KUBERNAUT" ]; then
        echo "Fetching kubernaut..."
        # curl -s -L -o "$KUBERNAUT" https://s3.amazonaws.com/datawire-static-files/kubernaut/$(curl -s https://s3.amazonaws.com/datawire-static-files/kubernaut/stable.txt)/kubernaut
        curl -s -L -o "$KUBERNAUT" https://s3.amazonaws.com/datawire-static-files/kubernaut/0.1.39/kubernaut
        chmod +x "$KUBERNAUT"
    fi
}

check_kubernaut_token () {
    if [ $("$KUBERNAUT" kubeconfig | grep -c 'Token not found') -gt 0 ]; then
        echo "You need a Kubernaut token. Go to"
        echo ""
        echo "https://kubernaut.io/token"
        echo ""
        echo "to get one, then run"
        echo ""
        echo "sh $ROOT/save-token.sh \"\$token\""
        echo ""
        echo "to save it before trying again."

        exit 1
    fi
}

get_kubernaut_cluster () {
    get_kubernaut
    check_kubernaut_token

    echo "Dropping old cluster"
    "$KUBERNAUT" discard

    echo "Claiming new cluster"
    "$KUBERNAUT" claim
    export KUBECONFIG=${HOME}/.kube/kubernaut
}
