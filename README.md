Ambassador
==========

Ambassador is an overseer for Envoy in Kubernetes deployments. You can tell it about new services you're creating and deleting, and it will update Envoy's configuration to match.

Ambassador is ALPHA SOFTWARE. In particular, at present it does not include an authentication mechanism, and it updates Envoy's configuration only with a specific request to do so -- be aware!

Running Ambassador
==================

If you clone this repository, you'll have access to the Kubernetes resource files `postgres.yaml`, `sds.yaml`, and `ambassador.yaml`.

Postgres
--------

Ambassador uses Postgres as its backing store. If you're already using Postgres in your cluster, great -- Ambassador assumes it can connect to `postgres:5432` as user `postgres`. It will create a database called `ambassador` for its use.

If you don't already have Postgres running, then

```
kubectl apply -f postgres.yaml
```

will create a Postgres deployment and service for you.

Ambassador-SDS and Ambassador
-----------------------------

Ambassador is the API gateway built atop Envoy; Ambassador-SDS is the Service Discovery Service that Ambassador relies on. Both are required. The easy way to start them is

```
kubectl apply -f sds.yaml,ambassador.yaml
```

Using Ambassador
================

Once running, use `kubectl get services` to find the IP address of the Ambassador. Call that `$AMBASSADORIP`. Then

```
curl http://$AMBASSADORIP/ambassador/health
```

will do a health check;

```
curl http://$AMBASSADORIP/ambassador/services
```

will get a list of all user-defined services Ambassador knows about;

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/url/prefix/here", "port": 80 }' \
      http://$AMBASSADORIP/ambassador/service/$servicename
```

will create a new service;

```
curl -XDELETE http://$AMBASSADORIP/ambassador/service/$servicename
```

will delete a service; and

```
curl -XPUT http://$AMBASSADORIP/ambassador/services
```

will update Envoy's configuration to match the currently-defined set of services.

