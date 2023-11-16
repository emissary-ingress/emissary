#!/hint/bash
set -e

busyprograms=(
    kubestatus
    watt
    apiext
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

# Copy installer support into /opt/image-build to be run at docker build for the
# production image. Then run the installers for the builder container.
# Note: When this (ambassador's) post-compile runs, it always runs first, and
# every other post-compile runs as well. So this is the place to recreate the
# /opt/image-build tree from scratch so the builder container stays valid.
sudo rm -rf /opt/image-build
sudo install -D -t /opt/image-build /buildroot/ambassador/build-aux/install.sh
sudo cp -a /buildroot/ambassador/build-aux/installers /opt/image-build/
sudo /opt/image-build/install.sh

# run any extra, local post-compile task
if [ -f post-compile.local.sh ] ; then
    bash post-compile.local.sh
fi
