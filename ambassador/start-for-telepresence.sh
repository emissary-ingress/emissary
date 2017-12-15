ln -sf $TELEPRESENCE_ROOT/var/run/secrets /var/run/secrets
export LC_ALL=C.UTF-8
export LANG=C.UTF-8
python3 setup.py develop
# python3 kubewatch.py sync /etc/ambassador-config /etc/envoy.json
AMBASSADOR_NO_DIAGD=true bash entrypoint.sh
