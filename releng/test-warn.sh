#!bash

if [ ! -r ".skip_test_warning" ]; then
    cat <<EOF


================================
YOU ARE ABOUT TO START AMBASSADOR'S TESTS, which rely on a
Kubernetes cluster. This is your cluster's information:

Cluster info:
EOF

    kubectl cluster-info

    cat <<EOF

Current context:
EOF

    kubectl config current-context

    cat <<EOF

WARNING: Your current Kubernetes context will be WIPED OUT by this test.

EOF

    while true; do
        read -p 'Is this really OK? (y/N) ' yn

        case $yn in
            [Yy]* ) break;;
            [Nn]* ) exit 1;;
            * ) echo "Please answer yes or no.";;
        esac
    done

    cat <<EOF

Great. To stop this warning, you can 'touch .skip_test_warning', but you
have to do that by hand.

Starting test run!
EOF
fi

exit 0
