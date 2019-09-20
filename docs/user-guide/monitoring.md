# Monitoring with Prometheus and Grafana

Prometheus is an open-source monitoring and alerting system. When used along with Grafana, we can create a dynamic dashboard for monitoring ingress into our Kubernetes cluster.

## Deployment

This guide will focus on deploying Prometheus and Grafana alongside Ambassador in Kubernetes using the [Prometheus Operator](https://github.com/coreos/prometheus-operator)

**Note:** Both Prometheus and Grafana can be deployed as standalone applications outside of Kubernetes. This process is well-documented within the website and docs for their respective projects. 

### Ambassador

Ambassador makes it easy to output Envoy-generated statistics to Prometheus. For the remainder of this guide, it is assumed that you have installed and configured Ambassador into your Kubernetes cluster, and that it is possible for you to modify the global configuration of the Ambassador deployment.

Starting with Ambassador `0.71.0`, Prometheus can scrape stats/metrics directly from Envoy's `/metrics` endpoint, removing the need to [configure Ambassador to output stats to StatsD](/user-guide/monitoring#statsd-exporter). 

The `/metrics` endpoint can be accessed internally via the Ambassador admin port (default 8877):

```
http(s)://ambassador:8877/metrics
```

or externally by creating a `Mapping` similar to below:

```yaml
apiVersion: ambassador/v1
kind: Mapping
name: metrics
prefix: /metrics
rewrite: ""
service: localhost:8877
```

**Note**: Since `/metrics` in an endpoint on Ambassador itself, the `service` field can just reference the admin port on localhost.

### Prometheus Operator

The [Prometheus Operator](https://github.com/coreos/prometheus-operator) for Kubernetes provides an easy way to manage your Prometheus deployment using Kubernetes-style resources with custom resource definitions (CRDs).

In this section, we will deploy the Prometheus Operator using the standard YAML files. You can also install it with [helm](/user-guide/monitoring#intall-with-helm) if you prefer. 

1. Deploy the Prometheus Operator

   To deploy the Prometheus Operator, you can clone the repository and follow the instructions in the README, or simply apply the published YAML with `kubectl`.

    ```
    kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/bundle.yaml
    ```

2. Deploy Prometheus by creating a `Prometheus` CRD

    First, create RBAC resources for your Prometheus instance

    ```
    kubectl apply -f https://www.getambassador.io/yaml/monitoring/prometheus-rbac.yaml
    ``` 

    Then, copy the YAML below, and save it in a file called `prometheus.yaml`

    ```yaml
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: prometheus
    spec:
      type: ClusterIP
      ports:
      - name: web
        port: 9090
        protocol: TCP
        targetPort: 9090
      selector:
        prometheus: prometheus
    ---
    apiVersion: monitoring.coreos.com/v1
    kind: Prometheus
    metadata:
      name: prometheus
    spec:
      ruleSelector:
        matchLabels:
          app: prometheus-operator
      serviceAccountName: prometheus
      serviceMonitorSelector:
        matchLabels:
          app: ambassador
      resources:
        requests:
          memory: 400Mi
    ```

    ```
    kubectl apply -f prometheus.yaml
    ```


3. Create a `ServiceMonitor`

   Finally, we need tell Prometheus where to scrape metrics from. The Prometheus Operator easily manages this using a `ServiceMonitor` CRD. To tell Prometheus to scrape metrics from Ambassador's `/metrics` endpoint, copy the following YAML to a file called `ambassador-monitor.yaml`, and apply it with `kubectl`.

    If you are running an Ambassador version higher than 0.71.0 and want to scrape metrics directly from the `/metrics` endpoint of Ambassador running in the `default` namespace:

    ```yaml
    ---
    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: ambassador-monitor
      namespace: monitoring
      labels:
        app: ambassador
    spec:
      namespaceSelector:
        matchNames:
        - default
      selector:
        matchLabels:
          service: ambassador-admin
      endpoints:
      - port: ambassador-admin
    ```

Prometheus is now configured to gather metrics from Ambassador.

#### Notes on the Prometheus Operator

The Prometheus Operator creates a series of Kubernetes Custom Resource Definitions (CRDs) for managing Prometheus in Kubernetes.

| Custom Resource Definition | Description |
| -------------------------- | ------------------------------------------------------------- |
| `AlertManager`             | An AlertManager handles alerts sent by the Prometheus server. |
| `PrometheusRule`           | Registers altering and reporting rules with Prometheus.       |
| `Prometheus`               | Creates a Prometheus instance.                                |
| `ServiceMonitor`           | Tells Prometheus where to scrape metrics from.                |

CoreOS has published a full [API reference](https://coreos.com/operators/prometheus/docs/latest/api.html) to these different CRDs.

### Grafana

Grafana is an open source graphing tool for plotting data points. Grafana allows you to create dynamic dashboards for monitoring your ingress traffic stats collected from Prometheus.

We have published a [sample dashboard](https://grafana.com/dashboards/10434) you can use for monitoring your ingress traffic. Since the stats from the `/metrics` and `/stats` endpoints are different, you will see a section in the dashboard for each use case.

**Note:** If you deployed the Prometheus Operator via the Helm Chart, a Grafana dashboard is created by default. You can use this dashboard or set `grafana.enabled: false` and follow the instructions below.

To deploy Grafana behind Ambassador: replace `{{AMBASSADOR_IP}}` with the IP address of your Ambassador service, copy the YAML below, and apply it with `kubectl`:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
      componenet: core
  template:
    metadata:
      labels:
        app: grafana
        component: core
    spec:
      containers:
      - image: grafana/grafana:6.2.0
        name: grafana-core
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 100m
            memory: 100Mi
          requests:
            cpu: 100m
            memory: 100Mi
        env:
          - name: GR_SERVER_ROOT_URL
            value: {{AMBASSADOR_IP}}/grafana
          - name: GF_AUTH_BASIC_ENABLED
            value: "true"
          - name: GF_AUTH_ANONYMOUS_ENABLED
            value: "false"
        readinessProbe:
          httpGet:
            path: /login
            port: 3000
        volumeMounts:
        - name: grafana-persistent-storage
          mountPath: /var
      volumes:
      - name: grafana-persistent-storage
        emptyDir: {}
```

Now, create a service and `Mapping` to expose Grafana behind Ambassador:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: monitoring
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: grafana-mapping
      prefix: /grafana/
      service: grafana.monitoring
spec:
  ports:
    - port: 80
      targetPort: 3000
  selector:
    app: grafana
    component: core
```

Now, access Grafana by going to `{AMBASSADOR_IP}/grafana/` and logging in with `username: admin` : `password: admin`. 

Import the [provided dashboard](https://grafana.com/dashboards/10434) by clicking the plus sign in the left side-bar, clicking `New Dashboard` in the top left, selecting `Import Dashboard`, and entering the dashboard ID(10434).

## Viewing Stats/Metrics

Above, you have created an environment where you have Ambassador as an API gateway, Prometheus scraping and collecting statistics output by Envoy about ingress into our cluster, and a Grafana dashboard to view these statistics. 

You can easily view a sample of these statistics via the Grafana dashboard at `{AMBASSADOR_IP}/grafana/` and logging in with the credentials above.

The example dashboard you installed above displays 'top line' statistics about the API response codes, number of connections, connection length, and number of registered services.

To view the full set of stats available to Prometheus you can access the Prometheus UI by running:

```
kubectl port-forward -n monitoring service/prometheus 9090
```
and going to http://localhost:9090/ from a web browser

In the UI, click the dropdown and see all of the stats Prometheus is able to scrape from Ambassador! 

The Prometheus data model is, at it's core, time-series based. Therefore, it makes it easy to represent rates, averages, peaks, minimums, and histograms. Review the [Prometheus documentation](https://prometheus.io/docs/concepts/data_model/) for a full reference on how to work with this data model.


---

## Additional Install Options

### Statsd Exporter

#### Ambassador

If running a pre-`0.71.0` version of Ambassador, you will need to configure Envoy to output stats to a separate collector before being scraped by Prometheus. You will use the [Prometheus StatsD Exporter](https://github.com/prometheus/statsd_exporter) to do this.

1. Deploy the StatsD Exporter in the `default` namespace

    ```
    kubectl apply -f https://getambassador.io/yaml/monitoring/statsd-sink.yaml
    ```

2. Configure Ambassador to output statistics to statsd

    In the Ambassador deployment, add the `STATSD_ENABLED` and `STATSD_HOST` environment variables to tell Ambassador where to output stats.

    ```yaml
    ...
            env:
            - name: AMBASSADOR_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace  
            - name: STATSD_ENABLED
              value: "true"
            - name: STATSD_HOST
              value: "statsd-sink.default.svc.cluster.local"
    ...  
    ``` 

Ambassador is now configured to output statistics to the Prometheus StatsD exporter.

#### ServiceMonitor

If you are scraping metrics from a `statsd-sink` deployment, you will configure the `ServiceMonitor` to scrape from that deployment.

```yaml
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: statsd-monitor
  namespace: monitoring
  labels:
    app: ambassador
spec:
  namespaceSelector:
    matchNames:
    - default
  selector:
    matchLabels:
      service: statsd-sink
  endpoints:
  - port: prometheus-metrics -->
```

### Prometheus Operator
#### Install with Helm

You can also use Helm to install Prometheus via the Prometheus Operator. The default [Helm Chart](https://github.com/helm/charts/tree/master/stable/prometheus-operator) will install Prometheus and configure it to monitor your Kubernetes cluster.

This section will focus on setting up Prometheus to scrape stats from Ambassador. Configuration of the Helm Chart and analysis of stats from other cluster components is outside of the scope of this documentation. 

1. Install the Prometheus Operator from the helm chart

    ```
    helm install -n prometheus stable/prometheus-operator
    ```

2. Create a `ServiceMonitor` 

    The Prometheus Operator Helm chart creates a Prometheus instance that is looking for `ServiceMonitor`s with `label: release=prometheus`.

    If you are running an Ambassador version higher than 0.71.0 and want to scrape metrics directly from the `/metrics` endpoint of Ambassador running in the `default` namespace:

    ```yaml
    ---
    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: ambassador-monitor
      namespace: monitoring
      labels:
        release: prometheus
    spec:
      namespaceSelector:
        matchNames:
        - default
      selector:
        matchLabels:
          service: ambassador-admin
      endpoints:
      - port: ambassador-admin
    ```

    If you are scraping metrics from a `statsd-sink` deployment:

    ```yaml
    ---
    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: statsd-monitor
      namespace: monitoring
      labels:
        release: prometheus
    spec:
      namespaceSelector:
        matchNames:
        - default
      selector:
        matchLabels:
          service: statsd-sink
      endpoints:
      - port: prometheus-metrics
    ```

Prometheus is now configured to gather metrics from Ambassador. 