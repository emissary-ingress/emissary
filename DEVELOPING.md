Developing AES
==============

AES is a complex piece of software with lots of integrations and
moving parts. Just being able to build the code and run tests is often
not sufficient to work efficiently on a given piece of the code. This
document functions as a central registry for how to **efficiently**
hack on any part of AES.

This guide is an extension of the [Ambassador OSS dev guide](https://github.com/datawire/ambassador/blob/master/DEVELOPING.md).
Please review that guide first for basic build and test instructions.

How do I hack on the UI?
------------------------

1. Ensure that you have a `ambassador.git` checkout next to your
   `apro.git` checkout, and make sure that it is in-sync with the
   `ambassador.commit` file in apro:

   ```sh
   (cd ../ambassador/ && git fetch && git checkout $(cat ../apro/ambassador.commit))
   ```

2. Run the sidecar locally with the local backend forwarding all
   snapshot requests to the backend in the cluster.

   ```sh
   DEV_WEBUI_SNAPSHOT_HOST=${MY_CLUSTER_HOST_OR_IP} \
   DEV_WEBUI_DIR=${PWD}/cmd/amb-sidecar/webui/bindata \
   POD_NAMESPACE=ambassador \
   DEV_AES_HTTP_PORT=8501 \
   DEV_WEBUI_PORT=9000 \
   go run ./cmd/ambassador amb-sidecar
   ```

3. Visit http://localhost:9000 in your browser

4. Hack away at the files in `${PWD}/cmd/amb-sidecar/webui/bindata/`. Refresh (or shift-refresh)
   your browser as necessary to get the updated files.

How do I hack on the UI using an IDE on a Mac (e.g. JetBrains WebStorm)?
------------------------------------------------------------------------

1. Ensure that you have a `ambassador.git` checkout next to your
   `apro.git` checkout, and make sure that it is in-sync with the
   `ambassador.commit` file in apro:

   ```sh
   (cd ../ambassador/ && git fetch && git checkout $(cat ../apro/ambassador.commit))
   ```

2. Set up JetBrains WebStorm. These set-up tasks only need to be done once:
   1. Configure [Live Edit](https://www.jetbrains.com/help/webstorm/live-editing.html).
   2. Configure [the Javscript debugger](https://www.jetbrains.com/help/webstorm/configuring-javascript-debugger.html)
   3. _(optional)_ Configure WebStorm to use [your Chrome user configuration](https://www.jetbrains.com/help/webstorm/configuring-browsers.html#enablingUseOfBrowsers)
   4. Add the [JetBrains extension to Chrome](https://chrome.google.com/webstore/detail/jetbrains-ide-support/hmhgeddbohgjknpmjagkdomcpobmllji).
      Note that you need to add the extension to the user being used by WebStorm. So if
      you didn't configure in #3, you have to open Chrome by running something from
      WebStorm, then install the extension in that instance of Chrome.
   5. Open a WebStorm project on `${PWD}/cmd/amb-sidecar/webui/bindata/edge_stack`
   6. Use Run > Edit Configurations.. to add a run configuration for `admin/index.html`
      with `?debug-backend=http://localhost:9000` at the end of the URL.

3. Run the sidecar locally with the local backend forwarding all
   snapshot requests to the backend in the cluster.

   ```sh
   DEV_WEBUI_WEBSTORM=1 \
   DEV_WEBUI_SNAPSHOT_HOST=<my-cluster-host-or-ip> \
   DEV_WEBUI_DIR=${PWD}/cmd/amb-sidecar/webui/bindata \
   POD_NAMESPACE=ambassador \
   DEV_AES_HTTP_PORT=8501 \
   DEV_WEBUI_PORT=9000 \
   go run ./cmd/ambassador amb-sidecar
   ```

4. Use "Run > Debug" to 'run' everything. This opens Chrome, attachs the debugger,
   opens index.html, etc.
   
5. The only awkward part of the dev loop at this time is that the security JWT
   is passed in through the URL, but the URL is defined in the run configuration.
   So you would need to update the run configuration with the JWT. However, for
   convenience, when you run with the `?debug-backend=` feature, there is an
   extra panel at the bottom of the login page with a button to enter the JWT.
   Use it by running `edgectl login`, then copying the url from the browser
   bar of the window that opens, closing that window, then clicking the "Enter
   URL+JWT" button and pasting the URL.

How do I hack on the UI without a cluster?
------------------------------------------

1. Ensure that you have a `ambassador.git` checkout next to your
   `apro.git` checkout, and make sure that it is in-sync with the
   `ambassador.commit` file in apro:

   ```sh
   (cd ../ambassador/ && git fetch && git checkout $(cat ../apro/ambassador.commit))
   ```

2. Run the sidecar locally:

   ```sh
   DEV_WEBUI_DIR=${PWD}/cmd/amb-sidecar/webui/bindata \
   POD_NAMESPACE=ambassador \
   DEV_AES_HTTP_PORT=8501 \
   DEV_WEBUI_PORT=9000 \
   go run ./cmd/ambassador amb-sidecar
   ```

3. To spoof cluster data, run:

   ```sh
   curl -X POST localhost:9000/_internal/v0/watt?push --data-binary @ui_devloop/snapshot.yaml
   ```

4. Visit http://localhost:9000 in your browser

5. Hack away at the files in `${PWD}/cmd/amb-sidecar/webui/bindata/`.
Refresh (or shift-refresh) your browser as necessary to get the updated files.

**NOTE:** You will need to re-do the spoofing each time you restart
the local sidecar.

How do I hack on the UI without building the code?
--------------------------------------------------

Run all these commands from the root of your apro checkout:

To run a stubbed out webui, in terminal:

1. Docker pull the aes image (with the right version): `docker pull quay.io/datawire/aes:<version>`.

2. Then run the following command (with the right version):

   ```sh
   docker run -it --rm \
       --volume=$(pwd)/cmd/amb-sidecar/webui/bindata:/ambassador/webui/bindata \
       --env=DEV_WEBUI_PORT=9000 --publish=9000:9000 \
       --entrypoint=/ambassador/sidecars/amb-sidecar \
       quay.io/datawire/aes:<version>
   ```

3. To spoof cluster data, run:

   ```sh
   curl -X POST localhost:9000/_internal/v0/watt?push --data-binary @ui_devloop/snapshot.yaml
   ```

4. Visit http://localhost:9000 in your browser

5. Hack away at the files in `${PWD}/cmd/amb-sidecar/webui/bindata/`.
Refresh (or shift-refresh) your browser as necessary to get the updated files.

**NOTE:** You will need to re-do the spoofing each time you restart
the local sidecar.

How does the UI work?
---------------------

Almost all user supplied ambassador inputs are CRDs and/or existing
kubernetes resources. (There are some minor exceptions in the form of
environment variables and files defined in the deployment. These
exceptions are one-time setup/bootstrap configuration.)

Ambassador communicates with users by watching for certain CRDs and
kubernetes resources to be defined, and by updating the status fields
of those resources to provide user feedback.

The UI is really just a way to render some/all of the ambassador
inputs graphically in a way that is helpful to users, as well as
supplying controls to allow a user to quickly produce new/updated yaml
manifests and either directly apply them to the cluster or download
them to check into git and/or apply by hand.

There are (primarily) two backend endpoints that the UI leverages:

/edge_stack/api/snapshot --> Returns the raw watt snapshot.
/edge_stack/api/apply --> Applies kubernetes yaml to the cluster.
/edge_stack/api/delete --> Deletes a kubernetes resource from the cluster.
