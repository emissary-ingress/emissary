# Inner dev-loop for the edge stack web UI

Run all these commands from the root of your apro checkout:

1. To run a stubbed out webui, in terminal 1:

   Select a Docker image containing amb-sidecar.  You can create one
   named `aes:latest` by running `make images`, or the latest RC can
   be downloaded from `quay.io/datawire-dev/aes:0.99.0-rc-latest`.

   Then run the following command (use the appropriate Docker image
   name instead of `aes:latest`):

   ```sh
   docker run -it \
       --volume=$(pwd)/cmd/amb-sidecar/webui/bindata:/ambassador/webui/bindata \
       --env=DEV_WEBUI_PORT=9000 --publish=9000:9000 \
       --entrypoint=/ambassador/sidecars/amb-sidecar \
       aes:latest
   ```

   Alternatively, if you have `go`:

    a. Ensure that you have a `ambassador.git` checkout next to your
       `apro.git` checkout, and make sure that it is in-sync with the
       `ambassador.commit` file in apro:

       ```sh
       (cd ../ambassador/ && git fetch && git checkout $(cat ../apro/ambassador.commit))
       ```

    b. Run the sidecar locally:

       ```sh
       DEV_WEBUI_DIR=${PWD}/cmd/amb-sidecar/webui/bindata APRO_HTTP_PORT=8501 DEV_WEBUI_PORT=9000 go run ./cmd/amb-sidecar
       ```

2. Visit http://localhost:9000 in your browser

3. Hack away at the files in `${PWD}/cmd/amb-sidecar/webui/bindata/`

4. To spoof cluster data, run:

   ```sh
   curl -X POST localhost:9000/_internal/v0/watt?push --data-binary @ui_devloop/snapshot.yaml
   ```

   This will load some interesting data into the sidecar. This can let
   you test out the UI with different data/states. Note there are
   other snapshots in the ui_devlooop directory.

4a. To get data directly from the cluster, use the DEV_WEBUI_SNAPSHOT_HOST environment variable:

       ```sh
       DEV_WEBUI_SNAPSHOT_HOST=<my-cluster-host-or-ip> DEV_WEBUI_DIR=${PWD}/cmd/amb-sidecar/webui/bindata APRO_HTTP_PORT=8501 DEV_WEBUI_PORT=9000 go run ./cmd/amb-sidecar
       ```

    The local backend will then forward all snapshot requests to the backend in the cluster.

5. Goto (3).

# UI Dev Big Picture

Almost all user supplied ambassador inputs are CRDs and/or existing
kubernetes resources. (There are some minor exceptions in the form of
environment variables and files defined in the deployment. These
exceptions are one-time setup/bootstrap configuration.)

Ambassador communicates with users by watching for certain CRDs and
kubernetes resources to be defined, and by updateing the status fields
of those resources to provide user feedback.

The UI is really just a way to render some/all of the ambassador
inputs graphically in a way that is helpful to users, as well as
supplying controls to allow a user to quickly produce new/updated yaml
manifests and either directly apply them to the cluster or download
them to check into git and/or apply by hand.

There are (primarily) two backend endpoints that the UI leverages:

/edge_stack/api/snapshot --> Returns the raw watt snapshot.
/edge_stack/api/apply --> Applies kubernetes yaml to the cluster.
