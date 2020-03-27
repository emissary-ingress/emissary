# Monitoring Ingress with Prometheus and Grafana

Prometheus is an open-source monitoring and alerting system. When used along with Grafana, you can create a dynamic dashboard for monitoring ingress into our Kubernetes cluster.

## Deployment

This guide will focus on deploying Prometheus and Grafana alongside Ambassador Edge Stack in Kubernetes using the [Prometheus Operator](https://github.com/coreos/prometheus-operator).

**Note:** Both Prometheus and Grafana can be deployed as standalone applications outside of Kubernetes. This process is well-documented within the website and docs within their respective projects.

### Ambassador Edge Stack

Ambassador Edge Stack makes it easy to output Envoy-generated statistics to Prometheus. For the remainder of this guide, it is assumed that you have installed and configured Ambassador Edge Stack into your Kubernetes cluster, and that it is possible for you to modify the global configuration of the Ambassador Edge Stack deployment.

Starting with Ambassador `0.71.0`, Prometheus can scrape stats/metrics directly from Envoy's `/metrics` endpoint, removing the need to [configure Ambassador Edge Stack to output stats to StatsD](#statsd-exporter-output-statistics-to-ambassador-edge-stack).

The `/metrics` endpoint can be accessed internally via the Ambassador Edge Stack admin port (default 8877):

```
http(s)://ambassador:8877/metrics
```

or externally by creating a `Mapping` similar to below:

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata: 
  name: metrics
spec:     
  prefix: /metrics
  rewrite: ""
  service: localhost:8877
```

**Note**: Since `/metrics` in an endpoint on Ambassador Edge Stack itself, the `service` field can just reference the admin port on localhost.

### Prometheus Operator with Standard YAML

In this section, we will deploy the Prometheus Operator using the standard YAML files. Alternatively, you can install it with [Helm](#prometheus-operator-with-helm) if you prefer.

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

   Finally, we need to tell Prometheus where to scrape metrics from. The Prometheus Operator easily manages this using a `ServiceMonitor` CRD. To tell Prometheus to scrape metrics from Ambassador Edge Stack's `/metrics` endpoint, copy the following YAML to a file called `ambassador-monitor.yaml`, and apply it with `kubectl`.

    If you are running an Ambassador version higher than 0.71.0 and want to scrape metrics directly from the `/metrics` endpoint of Ambassador Edge Stack running in the `ambassador` namespace:

    ```yaml
    ---
    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: ambassador-monitor
      labels:
        app: ambassador
    spec:
      namespaceSelector:
        matchNames:
        - ambassador
      selector:
        matchLabels:
          service: ambassador-admin
      endpoints:
      - port: ambassador-admin
    ```

Prometheus is now configured to gather metrics from Ambassador Edge Stack.

### Prometheus Operator with Helm

In this section, we will deploy the Prometheus Operator using Helm. Alternatively, you can install it with [kubectl YAML](#prometheus-operator-with-standard-yaml) if you prefer.

The default [Helm Chart](https://github.com/helm/charts/tree/master/stable/prometheus-operator) will install Prometheus and configure it to monitor your Kubernetes cluster.

This section will focus on setting up Prometheus to scrape stats from Ambassador Edge Stack. Configuration of the Helm Chart and analysis of stats from other cluster components is outside of the scope of this documentation.

1. Install the Prometheus Operator from the helm chart

    ```	
    helm install -n prometheus stable/prometheus-operator
    ```

2. Create a `ServiceMonitor`

    The Prometheus Operator Helm chart creates a Prometheus instance that is looking for `ServiceMonitor`s with `label: release=prometheus`.

    If you are running an Ambassador version higher than 0.71.0 and want to scrape metrics directly from the `/metrics` endpoint of Ambassador Edge Stack running in the `default` namespace:

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

Prometheus is now configured to gather metrics from Ambassador Edge Stack.

#### Prometheus Operator CRDs

The Prometheus Operator creates a series of Kubernetes Custom Resource Definitions (CRDs) for managing Prometheus in Kubernetes.

| Custom Resource Definition | Description |
| -------------------------- | ------------------------------------------------------------- |
| `AlertManager`             | An AlertManager handles alerts sent by the Prometheus server. |
| `PrometheusRule`           | Registers altering and reporting rules with Prometheus.       |
| `Prometheus`               | Creates a Prometheus instance.                                |
| `ServiceMonitor`           | Tells Prometheus where to scrape metrics from.                |

CoreOS has published a full [API reference](https://coreos.com/operators/prometheus/docs/latest/api.html) to these different CRDs.

### Grafana

Grafana is an open-source graphing tool for plotting data points. Grafana allows you to create dynamic dashboards for monitoring your ingress traffic stats collected from Prometheus.

We have published a [sample dashboard](https://grafana.com/grafana/dashboards/4698) you can use for monitoring your ingress traffic. Since the stats from the `/metrics` and `/stats` endpoints are different, you will see a section in the dashboard for each use case.

**Note:** If you deployed the Prometheus Operator via the Helm Chart, a Grafana dashboard is created by default. You can use this dashboard or set `grafana.enabled: false` and follow the instructions below.

To deploy Grafana behind Ambassador Edge Stack: replace `{{AMBASSADOR_IP}}` with the IP address or DNS name of your Ambassador Edge Stack service, copy the YAML below, and apply it with `kubectl`:

```yaml
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: grafana
  labels:
    app: grafana
    component: core
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
      component: core
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: grafana
        component: core
      annotations:
        sidecar.istio.io/inject: 'false'
    spec:
      volumes:
        - name: data
          emptyDir: {}
      containers:
        - name: grafana
          image: 'grafana/grafana:6.4.3'
          ports:
            - containerPort: 3000
              protocol: TCP
          env:
            - name: GF_SERVER_ROOT_URL
              value: {{AMBASSADOR_IP}}/grafana
            - name: GRAFANA_PORT
              value: '3000'
            - name: GF_AUTH_BASIC_ENABLED
              value: 'false'
            - name: GF_AUTH_ANONYMOUS_ENABLED
              value: 'true'
            - name: GF_AUTH_ANONYMOUS_ORG_ROLE
              value: Admin
            - name: GF_PATHS_DATA
              value: /data/grafana
          resources:
            requests:
              cpu: 10m
          volumeMounts:
            - name: data
              mountPath: /data/grafana
          readinessProbe:
            httpGet:
              path: /api/health
              port: 3000
              scheme: HTTP
            timeoutSeconds: 1
            periodSeconds: 10
            successThreshold: 1
            failureThreshold: 3
          imagePullPolicy: IfNotPresent
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
spec:
  ports:
    - port: 80
      targetPort: 3000
  selector:
    app: grafana
    component: core
```

Now, create a service and `Mapping` to expose Grafana behind Ambassador Edge Stack:

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata: 
  name: grafana
spec:     
  prefix: /grafana/
  service: grafana.{{GRAFANA_NAMESPACE}}
```

Now, access Grafana by going to `{AMBASSADOR_IP}/grafana/` and logging in with `username: admin` : `password: admin`.

Import the [provided dashboard](https://grafana.com/dashboards/10434) by clicking the plus sign in the left side-bar, clicking `New Dashboard` in the top left, selecting `Import Dashboard`, and entering the dashboard ID(10434).

## Viewing Stats/Metrics

Above, you have created an environment where Ambassador is handling ingress traffic, Prometheus is scraping and collecting statistics from Envoy, and Grafana is displaying these statistics in a dashboard.

You can easily view a sample of these statistics via the Grafana dashboard at `{AMBASSADOR_IP}/grafana/` and logging in with the credentials above.

The example dashboard you installed above displays 'top line' statistics about the API response codes, number of connections, connection length, and number of registered services.

To view the full set of stats available to Prometheus you can access the Prometheus UI by running:

```
kubectl port-forward -n monitoring service/prometheus 9090
```

and going to `http://localhost:9090/` from a web browser

In the UI, click the dropdown and see all of the stats Prometheus is able to scrape from Ambassador Edge Stack.

The Prometheus data model is, at its core, time-series based. Therefore, it makes it easy to represent rates, averages, peaks, minimums, and histograms. Review the [Prometheus documentation](https://prometheus.io/docs/concepts/data_model/) for a full reference on how to work with this data model.

---

## Additional Install Options

### StatsD Exporter: Output Statistics to Ambassador Edge Stack

If running a pre-`0.71.0` version of Ambassador, you will need to configure Envoy to output stats to a separate collector before being scraped by Prometheus. You will use the [Prometheus StatsD Exporter](https://github.com/prometheus/statsd_exporter) to do this.

1. Deploy the StatsD Exporter in the `default` namespace

    ```
    kubectl apply -f https://www.getambassador.io/yaml/monitoring/statsd-sink.yaml
    ```

2. Configure Ambassador Edge Stack to output statistics to `statsd`

    In the Ambassador Edge Stack deployment, add the `STATSD_ENABLED` and `STATSD_HOST` environment variables to tell Ambassador Edge Stack where to output stats.

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

Ambassador Edge Stack is now configured to output statistics to the Prometheus StatsD exporter.

#### ServiceMonitor

If you are scraping metrics from a `statsd-sink` deployment, you will configure the `ServiceMonitor` to scrape from that deployment.

```yaml
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: statsd-monitor
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
  - port: prometheus-metrics
```
