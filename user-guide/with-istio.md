# Ambassador and Istio

---

Are you looking to run Ambassador without Istio? You probably want to check out the [Getting Started](getting-started.md) guide for Ambassador alone.

---

Ambassador is an API Gateway for microservices. [Istio](https://istio.io/) is a service mesh for microservices. Both use [Envoy](https://lyft.github.io/envoy/) for the heavy lifting.

Given the use of Envoy, there's a good amount of overlap between the two. In particular, we expect to be able to bring more of Ambassador's functionality into Istio over time -- but for now, using Ambassador as an API gateway fronting an Istio mesh is the simplest way to get an integrated service mesh that handles external traffic. Ambassador takes care of URL rewriting and managing the `Host` header on the way into your microservices, two things that can be otherwise quite irritating for your microservice developers.

## Caveats

It's still early days for both Ambassador and Istio, so **at present we have not tested Ambassador with the Istio Auth feature**. That'll happen soon, never fear.

We also assume that you've already gotten a Kubernetes cluster set up with Istio: Ambassador relies on being able to see Istio already running when it's launched, so **it will not work** to launch Ambassador, then Istio. If you don't already have Istio running, check out their [instructions for installing Istio](https://istio.io/docs/tasks/installing-istio.html).

Make sure to remove Istio's default ingress controller, as we are about to replace it. If `kubectl get ingress` shows you an ingress controller:

```shell
NAME      HOSTS     ADDRESS          PORTS     AGE
gateway   *         104.154.161.38   80        10m
```

make sure you remove it (`kubectl delete ingress gateway`) before proceeding.

## Starting Ambassador

Once Istio is running (minus its default ingress controller -- see above), you can start Ambassador running as follows:

```shell
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/istio/ambassador.yaml
```

That will launch Ambassador and configure the Istio ingress controller to route all HTTP requests to Ambassador for routing. At this point:

- Istio's ingress controller is your route into the cluster.
- Ambassador will handle traffic routing at the edge.
- Istio's mesh will handle all inter-service communications.

`kubectl get pods` should show you something like this:

```shell
NAME                             READY     STATUS    RESTARTS   AGE
ambassador-4101501082-pgmdg      2/2       Running   0          29m
astore-601808212-107x7           1/1       Running   0          29m
istio-egress-366830726-lkf25     1/1       Running   0          29m
istio-ingress-538562089-wff9n    1/1       Running   0          29m
istio-manager-1957901723-jq5p3   2/2       Running   0          29m
istio-mixer-1989441528-dw6z6     1/1       Running   0          29m
```

where `ambassador` and `astore` are the pods that Ambassador needs to run, and the `istio-` pods form the backplane of the Istio service mesh.

## A Test Application

We'll test this by deploying a really simple application called `micromaze`, which comprises three microservices: the `usersvc`, the `gruesvc`, and the `mazesvc`. In the world of this app, users and grues wander around a maze, but the important point here is simply that the `mazesvc` has to talk to the `usersvc` and the `gruesvc`, both of which in turn have to talk to a Postgres database.

Bare-bones versions of all three apps live on GitHub in our [micromaze](https://github.com/datawire/micromaze). Since Datawire has already published Docker images for them on DockerHub, you don't need to clone that repo unless you're curious about the microservices, or you want to rebuild images locally for some reason.

### Deploying the Microservices

Getting the three microservices hooked into the Istio service mesh isn't quite as simple as just getting them deployed into Kubernetes. We need the service, yes, but each instance of each service also needs an Envoy running alongside it, participating in the Istio mesh. This would be pretty painful to do by hand, so Istio provides tooling to automate it.

However, where `kubectl` can directly use definition files via GitHub URLs, as we do above, Istio's tooling need local files. So we'll start by downloading the YAML file we'll need:

```shell
curl -o micromaze.yaml https://raw.githubusercontent.com/datawire/ambassador/master/istio/micromaze.yaml
```

(`micromaze.yaml` is built from four smaller YAML files, which you can see in the [micromaze](https://github.com/datawire/micromaze) repo if you're curious.)

Once you've downloaded `micromaze.yaml`, you can deploy the `micromaze` app into the Istio mesh using `bash`:

```shell
export MANAGER_HUB="docker.io/istio"
export MANAGER_TAG="0.1.2"
kubectl apply -f <(istioctl kube-inject -f micromaze.yaml)
```

`istioctl kube-inject` reads the YAML definition handed to it and outputs a modified version that includes an appropriately-configured Envoy sidecar; we use it here with the `<()` construct of `bash` to pass that modified output to `kubectl apply` as a file. `MANAGER_HUB` and `MANAGER_TAG` are needed for `istioctl kube-inject` to know what exactly to inject.

Once this is done, `kubectl get pods` should show quite a few pods running:

```shell
NAME                             READY     STATUS    RESTARTS   AGE
ambassador-4101501082-pgmdg      2/2       Running   0          29m
astore-601808212-107x7           1/1       Running   0          29m
gruesvc-934809961-4rwd0          2/2       Running   0          16m
istio-egress-366830726-lkf25     1/1       Running   0          29m
istio-ingress-538562089-wff9n    1/1       Running   0          29m
istio-manager-1957901723-jq5p3   2/2       Running   0          29m
istio-mixer-1989441528-dw6z6     1/1       Running   0          29m
mazesvc-3997176150-phrxr         2/2       Running   0          16m
postgres-3331715136-k7c57        2/2       Running   0          16m
usersvc-2967736717-03mgv         2/2       Running   0          16m
```

### Configuring Ambassador

At this point Ambassador has no mappings, and we need to define some. We'll use `kubectl port-forward` to get access to Ambassador's administrative interface:

```shell
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
```

Once that's done, `localhost:8888` is where you can talk to the Ambassador's administrative interface. Let's start with a basic health check of Ambassador itself:

```shell
curl http://localhost:8888/ambassador/health
```

which should give something like this if all is well:

```json
{
  "hostname": "ambassador-4101501082-pgmdg",
  "msg": "ambassador health check OK",
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "0.10.3"
}
```

We need to map the `/maze/` resource to our `mazesvc`, which needs a POST request:

```shell
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/maze/", "service": "mazesvc", "rewrite": "/maze/" }' \
      http://localhost:8888/ambassador/mapping/maze_map
```

and after that, you can read back and see that the mapping is there:

```shell
curl http://localhost:8888/ambassador/mappings
```

which should show you something like

```json
{
  "count": 1,
  "hostname": "ambassador-4101501082-pgmdg",
  "mappings": [
    {
      "name": "maze_map",
      "prefix": "/maze/",
      "rewrite": "/maze/",
      "service": "mazesvc"
    }
  ],
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "0.10.3"
}
```

We won't map any other services -- the `mazesvc` is meant to be the only one exposed to the world. Note, though, the `rewrite` rule above: the `mazesvc` expects to see resources rooted at `/`, not at `/maze/`. This sort of relative-path ability makes life much easier for microservice developers.

### Using the Istio Ingress

To talk to the `mazesvc` we need to go through the the Istio ingress controller: it's our path into the mesh at this point, which means that we need to figure out how to talk to it. Sadly, this depends a bit on where your cluster is running, but `kubectl get ingress -o wide` should show you what you need to know in most cases:

```shell
$ kubectl get ingress -o wide
NAME             HOSTS     ADDRESS         PORTS     AGE
simple-ingress   *         35.184.167.177   80        1h
```

In this case (running on Google Container Engine) we'd set

```shell
export ISTIO_URL=http://35.184.167.177
```

(On Minikube, you'll probably do better with `minikube service --url istio-ingress`.)

Once that's done, you should be able to perform a simple health check of the `mazesvc` itself with

```shell
curl ${ISTIO_URL}/maze/health
```

and you should see something like

```json
{
  "hostname": "mazesvc-3997176150-q1wn0",
  "msg": "mazesvc health check OK",
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "0.2.0"
}
```

## Next steps

At this point, you've set up an Ambassador API Gateway into an Istio mesh, and you can proceed with simple, self-service path-based routing into your microservices, while also getting the advantages of Istio's service mesh, including its monitoring and tracing. We'll be working more on integrating Ambassador's functionality with Istio going forward; in the immediate term, they can already complement each other and make life easier on microservices developers.
