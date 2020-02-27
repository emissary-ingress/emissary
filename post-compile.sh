#!/hint/bash
set -e

# Create symlinks to the multi-call binary
ln -sf /buildroot/bin/ambassador /buildroot/bin/ambex
ln -sf /buildroot/bin/ambassador /buildroot/bin/kubestatus
ln -sf /buildroot/bin/ambassador /buildroot/bin/watt

# Stuff in /opt/ambassador/bin in the builder winds up in /usr/local/bin in the
# production image.
sudo install -D -t /opt/ambassador/bin/ /buildroot/bin/ambassador
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/ambex
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/kubestatus
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/watt
sudo install /buildroot/bin/capabilities_wrapper /opt/ambassador/bin/wrapper
