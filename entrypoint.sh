/usr/bin/python3 /application/envoy-restarter.py /etc/envoy-restarter.pid /application/envoy-wrapper.sh &
/usr/bin/python3 /application/ambassador.py /application/envoy-template.json /etc/envoy.json /etc/envoy-restarter.pid
