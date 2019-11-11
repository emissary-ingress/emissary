# Inner dev-loop for the edge stack web UI

Run all these commands from the root of your apro checkout:

1. To run a stubbed out webui, in terminal 1:

   Select a Docker image containing amb-sidecar.  You can create on
   named `aes:latest` by running `make images`, or the latest RC can
   be downloaded from `quay.io/datawire-dev/aes:0.10.0-rc-latest`.
   
   Ensure that your `KUBECONFIG` environmet variable is set.

   Then run the following command (use the appropriate Docker image
   name instead of `aes:latest`):

   ```sh
   docker run \
       --volume=$KUBECONFIG:/root/.kube/config \
       --volume=$PWD/cmd/amb-sidecar/webui/bindata:/ambassador/webui/bindata \
       --env=DEV_WEBUI_PORT=9000 --publish=9000:9000 \
       --entrypoint=/ambassador/sidecars/amb-sidecar \
       aes:latest
   ```

2. Visit http://localhost:9000 in your browser

3. Hack away at the files in `${PWD}/cmd/amb-sidecar/webui/bindata/`

4. To spoof cluster data, run:

   ```sh
   curl -X POST localhost:9000/_internal/v0/watt?push --data-binary @ui_devloop/snapshot.yaml
   ```

   This will load some interesting data into the sidecar. This can let
   you test out the UI with different data/states.

5. Goto (4).
