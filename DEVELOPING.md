This is an integration of the lyft ratelimit service into a formfactor
suitable for ambassador. This means:

 - deploying it beside ambassador as a sidecar
 - using CRDs to define it's configuration
 - supplying a basic controller to reload on config changes

The intention (for now) is to make only very minor codechanges to the
lyft ratelimit service itself, and so the Makefile pulls in a (pegged)
version of the lyft dependency that is very lightly patched. (See
comments in the Makefile around the `make diff` target for more
details.)

To get started:

1. Type `make shell` in order to start a dev shell. This will acquire
   a kubernaut cluster and launch a shell with KUBECONFIG setup to point
   to the acquired cluster.

2. Type `make manifests`. This will spin up ambassador, redis, and the
   ratelimit service. It will take a while the first time. It will be
   quicker subsequent times. You need to do this whenever you modify
   the k8s yaml.

3. Start teleproxy from a dev shell (teleproxy will be automatically
   installed into your dev shell environment) and then start another
   dev shell. The remaining steps all assume teleproxy is running and
   you are in another unoccupied dev shell.

To query the ratelimit service in the cluster:

1. Type `make compile` in order to build the ratelimit binaries. If
   you are in a dev shell, these binaries will be in your path:

   - ratelimit: the ratelimit service itself
   - ratelimit_client: a client for querying the ratelimit service
   - ratelimit_config_check

2. Run: `ratelimit_client -dial_string ratelimit:81 -domain mongo_cps -descriptors database=default`

To modify the ratelimit service config in the cluster:

1. Edit the configmap in k8s/ambassador.yaml. Note that all the config
   values need to start with "config." for the ratelimit service to
   see them (it filters out any config files that do not start with
   "config.").

2. Run `make manifests`.

3. Delete the ambassador pod (in order to get the config to reload).

To see the descriptors that ambassador produces:

1. `curl ambassador/rl/`

2. Look at the logs of the ratelimit container in the ambassador pod.

To run/query the ratelimit service locally:

1. Type `make run`. (The config will be loaded from the config/ directory.)

2. Run: `ratelimit_client -domain mongo_cps -descriptors database=default` (Note the dial_string is omitted)

3. Edit the config files in the config/ directory and restart your
   `make run`. Note that the files in the `config/` directory need to
   start with `config.` in order to be seen by the ratelimit service.

Building a docker image (XXX this needs to be parameterized):

1. docker build . -t rschloming/rl:<n>
2. docker push rschloming/rl:<n>
