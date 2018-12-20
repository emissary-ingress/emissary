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

1. Type `make deploy`. This will build a container, acquire a
   kubernaut cluster, and spin up ambassador, redis, and the ratelimit
   service. It will take a while the first time. It will be quicker
   subsequent times.

2. Type `make proxy` to start teleproxy. This will start teleproxy in
   the background. To stop it at any point type `make unproxy`.

The remaining steps all assume teleproxy is running. To query the
ratelimit service in the cluster:

1. Type `make lyft-build` in order to build the ratelimit binaries:

   - ratelimit: the ratelimit service itself
   - ratelimit_client: a client for querying the ratelimit service

2. Run: `./ratelimit_client -dial_string ratelimit:81 -domain test -descriptors a=b`

To modify the ratelimit service config in the cluster:

1. Edit the `k8s/limits.yaml` or add your own limits in another file
   underneath `k8s`.

2. Run `make apply`.

To see the descriptors that ambassador produces:

1. `curl ambassador/rl/`

2. Look at the logs of the ratelimit container in the ambassador pod.
