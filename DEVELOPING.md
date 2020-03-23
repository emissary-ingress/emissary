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

How do I hack on the AES metrics reporting to Metriton?
-------------------------------------------------------

AES collects usage metrics and sends them to Metriton, our backend metrics 
wrangler and database. You might hear the term "scout" as the initial version 
of Metriton was called the "Scout API".

In AES, different metrics data points are sent from both the A/OSS python 
component (see `ambscout.py`) and the `phonehome.go` library. This guide is 
only concerned with the Golang phonehome.go specific to AES.

1. Ensure the `SCOUT_DISABLE` environment variable is NOT SET in your 
ambassador deployment.

2. In `lib/metriton/phonehome.go`, you might want to change the following 
constants:
   - `phoneHomeEveryPeriod` --> Normally, AES would PhoneHome every 12 hours. 
   For testing purposes, you might want to run this routine every 5 minutes 
   for example. (5 * time.Minute)
   - `metritonEndpoint` --> Instead of sending metrics directly to the 
   production Metriton (and thus polluting production metrics data), you might 
   want to change the endpoint to `https://kubernaut.io/beta/scout`.
   
   Watch out not to commit any unintended change to these constants! 

Apart from reporting license information, AES will report usage data of 
licensed-features usage in this format:
   
   ```json
    {
       "id":"unregistered",
       "contact":"",
       "features":[
          { "name":"authfilter-service", "limit":5, "usage":0, "max_usage":1 },
          { "name":"devportal-services", "limit":5, "usage":1, "max_usage":1 },
          { "name":"ratelimit-service", "limit":5, "usage":0, "max_usage":5 }
       ],
       "component":"ambassador-sidecar",
       "user_agent":"Go-http-client/1.1"
    }
   ```

You may use the following manifests to easily get a working environment 
allowing you to use licensed features and generate some usage data points:

1. authfilter-service

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Filter
   metadata:
     name: example-jwt-filter
     namespace: ambassador
   spec:
     JWT:
       insecureTLS: true
       jwksURI: "https://getambassador-demo.auth0.com/.well-known/jwks.json"
       validAlgorithms:
         - "none"
   ---
   apiVersion: getambassador.io/v2
   kind: FilterPolicy
   metadata:
     name: auth-filter-policy
     namespace: ambassador
   spec:
     rules:
       - host: "*"
         path: /auth-filter-policy/
         filters:
           - name: "example-jwt-filter"
   ```
   
   Send any request to `/auth-filter-policy/` to increment usage of the 
   `authfilter-service` feature. This feature is tracking Requests Per Second
    usage (RPS).

2. ratelimit-service

   ```yaml
    apiVersion: getambassador.io/v2
    kind: Mapping
    metadata:
      name: backend-rate-limit
      namespace: ambassador
    spec:
      prefix: /backend-rate-limit/
      service: quote
      labels:
        ambassador:
          - request_label_group:
              - backend
    ---
    apiVersion: getambassador.io/v2
    kind: RateLimit
    metadata:
      name: backend-rate-limit
      namespace: ambassador
    spec:
      domain: ambassador
      limits:
        - pattern: [{generic_key: backend}]
          rate: 30
          unit: minute
   ```
   
   Send any request to `/backend-rate-limit/` to increment usage of the 
   `ratelimit-service` feature. This feature is tracking Requests Per Second 
   usage (RPS).

3. devportal-services

   ```yaml
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: petstore
      namespace: ambassador
    spec:
      ports:
        - name: backend
          port: 8080
          targetPort: 8080
      selector:
        app: petstore
    ---
    apiVersion: getambassador.io/v2
    kind: Mapping
    metadata:
      name: petstore
      namespace: ambassador
    spec:
      prefix: /petstore/
      service: petstore:8080
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: petstore
      namespace: ambassador
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: petstore
      strategy:
        type: RollingUpdate
      template:
        metadata:
          labels:
            app: petstore
        spec:
          containers:
            - name: backend
              image: agervais/petstore-backend:latest
              imagePullPolicy: Always
              ports:
                - name: http
                  containerPort: 8080
   ```

   This feature is tracking the number of published services with API 
   documentation (Count).

If you want to dig deeper, you may want to inspect Redis, as all rate-limiting 
data points are stored in Redis

   ```sh
   # Exec into the running Redis pod
   k exec -it ambassador-redis-[POD ID] -n ambassador -- /bin/sh
   
   # Start the redis client, which will open a redis> prompt.
   redis-cli

   # Start inspecting the content of the data store.
   keys *
   get ratelimit-service-m
   ttl ratelimit-service-m
   del ratelimit-service-m
   ```

How do I debug the OAuth browser tests?
---------------------------------------

Some of the OAuth filter tests run a headless web browser.  These
browser tests would normally be very difficult to debug.  However, to
make things easier, they record a video of the browser session, so you
can see what happened.  The video file is saved at

    ./tests/cluster/go-test/filter-oauth2/testdata/TESTNAME.webm

You can access this file normally when run locally.  For CI runs, it's
saved as a build artifact, so you can access that directory by
going to the "Artifacts" tab for the CircleCI build.

Often, those video files are enough to tell you what's going wrong.

For more advanced debugging, temporarily editing the file
`./tests/cluster/go-test/filter-oauth2/testdata/run.js` for your test
runs can be very useful.  Common modifications are

 - uncomment the `//headless: false,` line, so you can see things live
 - comment out the `browser.close()` line, so you can poke around in
   the browser after the test finishes
 - uncomment the `//await writeFile("/tmp/f.html", await
   browsertab.content());` line so you can inspect the DOM of the
   final page.

How do I add new CRD types?
---------------------------

Well, you should probably start by writing a spec for the CRD.  Should
you do that in Protobuf at `ambassador.git/api/`?  Should you do that
in JSON Schema at `ambassador.git/python/schemas/`?  Should you do
that in Go structs at `apro.git/apis/`?  Should you just say "YOLO"
and define it as a Go struct in the package where you consume it?  Who
knows, we get conflicting answers whenever we try to settle on one.

OK, so you somehow figured out how to get the code to understand and
listen for the CRD.  Now you need to add it to the YAML:

 1. Define the CRD.
    - for OSS CRDs, add it to each of the following files:
      * `ambassador.git/docs/yaml/ambassador/ambassador-crds.yaml`
      * `ambassador.git/docs/yaml/ambassador/ambassador-knative.yaml`
      * `ambassador.git/docs/yaml/ambassador/ambassador-rbac-prometheus.yaml`
    - for AES-only CRDs, add it to the following file:
      * `apro.git/k8s-aes-src/00-aes-crds.yaml`
 2. Update the RBAC.
    - If you need to adjust the OSS RBAC:
      1. Edit the `ClusterRole` in the following files:
         + `ambassador.git/docs/yaml/ambassador/ambassador-knative.yaml`
         + `ambassador.git/docs/yaml/ambassador/ambassador-rbac-prometheus.yaml`
         + `ambassador.git/docs/yaml/ambassador/ambassador-rbac.yaml`
         + `ambassador.git/python/tests/manifests/rbac_cluster_scope.yaml`
      2. Edit the `Role` in the following files:
         + `ambassador.git/python/tests/manifests/rbac_namespace_scope.yaml`
      3. Also edit the AES RBAC (below) correspondingly
    - If you need to adjust the AES RBAC, edit:
      + `apro.git/k8s-aes-src/01-aes.yaml` (edit the `ClusterRole`)
      + `apro.git/tests/pytest/manifests/rbac_cluster_scope.yaml` (edit the `ClusterRole`)
      + `apro.git/tests/pytest/manifests/rbac_namespace_scope.yaml`
        * You should mostly just be changing the `Role` (not the
          `ClusterRole`).  However, if you're not handling
          `AMBASSADOR_SINGLE_NAMESPACE` in client-go (as you're probably
          not if you're using `github.com/datawire/ambassador/pkg/k8s`
          directly instead of using WATT), then you also need to add
          get/list/watch for it to the `ClusterRole`.
 3. Update generated files.
    1. If you made any changes in `ambassador.git`, update
      `apro.git/ambassador.commit`.
    2. In `apro.git`, run `make update-yaml-locally`.
