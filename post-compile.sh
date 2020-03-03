#!/hint/bash
set -e

# Create symlinks to the multi-call binary so the original names can be used in
# the builder shell easily (from the shell PATH).
ln -sf /buildroot/bin/ambassador /buildroot/bin/ambex
ln -sf /buildroot/bin/ambassador /buildroot/bin/kubestatus
ln -sf /buildroot/bin/ambassador /buildroot/bin/watt

# Also note there is a different ambassador binary, written in Python, that
# shows up earlier in the shell PATH:
#   $ type -a ambassador
#   ambassador is /usr/bin/ambassador
#   ambassador is /buildroot/bin/ambassador

# Stuff in /opt/ambassador/bin in the builder winds up in /usr/local/bin in the
# production image.
sudo install -D -t /opt/ambassador/bin/ /buildroot/bin/ambassador
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/ambex
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/kubestatus
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/watt
sudo install /buildroot/bin/capabilities_wrapper /opt/ambassador/bin/wrapper
