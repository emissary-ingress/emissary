#!/hint/bash
set -e

busyprograms=(
    kubestatus
)
sudo install -D -t /opt/ambassador/bin/ /buildroot/bin/busyambassador
for busyprogram in "${busyprograms[@]}"; do
    # Create symlinks to the multi-call binary so the original names can be used in
    # the builder shell easily (from the shell PATH).
    ln -sf /buildroot/bin/busyambassador /buildroot/bin/"$busyprogram"
    # Stuff in /opt/ambassador/bin in the builder winds up in /usr/local/bin in the
    # production image.
    sudo ln -sf /opt/ambassador/bin/busyambassador /opt/ambassador/bin/"$busyprogram"
done

sudo install /buildroot/bin/capabilities_wrapper /opt/ambassador/bin/wrapper
sudo install /buildroot/bin/apiext /opt/ambassador/bin/apiext
