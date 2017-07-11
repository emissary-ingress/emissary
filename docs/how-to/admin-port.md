# Administration Port

Ambassador's admin interface is reachable over port 8888. This port is deliberately not exposed with a Kubernetes service; you'll need to use `kubectl port-forward` to reach it:

```
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
```

Once that's done, you can use the admin interface for health checks, statistics, [mappings](mappings.md#mappings), [modules](mappings.md#modules), and [consumers](mappings.md#consumers).

### Health Checks and Stats

```
curl http://localhost:8888/ambassador/health
```

will do a health check;

```
curl http://localhost:8888/ambassador/mapping
```

will get a list of all the resources that Ambassador has mapped; and

```
curl http://localhost:8888/ambassador/stats
```

will return a JSON dictionary containing a `stats` dictionary with statistics about resources that Ambassador presently has mapped. Most notably, `stats.mappings` contains basic health information about the mappings to which Ambassador is providing access:

- `stats.mappings.<mapping-name>.healthy_members` is the number of healthy back-end systems providing the mapped service;
- `stats.mappings.<mapping-name>.upstream_ok` is the number of requests to the mapped resource that have succeeded; and
- `stats.mappings.<mapping-name>.upstream_bad` is the number of requests to the mapped resource that have failed.

