/usr/bin/python3 /application/ambassador.py /application/envoy-template.json /application/envoy.json &
/usr/bin/python3 /application/hot-restarter.py /application/envoy-wrapper.sh
