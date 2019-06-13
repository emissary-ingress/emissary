# Monitoring with Prometheus and Grafana

Prometheus is an open-source monitoring and alerting system. When used along with Grafana, we can create a dynamic dashboard for monitoring ingress into our Kubernetes cluster.

## Deployment

This guide will focus on deploying Prometheus and Grafana alongside Ambassador in Kubernetes in a new namespace named `Monitoring`. 

**Note:** Both Prometheus and Grafana can be deployed as standalone applications outside of Kubernetes. This process is well documented in the documentation for their individual projects. 

### Ambassador

Ambassador makes it easy to output Envoy generated statistics to Prometheus. 

To Envoy statistics in a way Prometheus can ingest, we need to use the [Prometheus StatsD Exporter](https://github.com/prometheus/statsd_exporter). 

1. Deploy the StatsD Exporter in the `Monitoring` namespace

    ```
    kubectl apply -f https://getambassador.io/yaml/monitoring/statsd-sink.yaml
    ```

    This will create the `Monitoring` namespace and the `statsd-sink` service and deployment for exposing the StatsD Exporter tool to Ambassador.

2. Configure Ambassador to output statistics to statsd

    In the Ambassador deployment, add the `STATSD_ENABLED` and `STATSD_HOST` environment variables to tell Ambassador where to output statistics.

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
              value: "statsd-sink.monitoring.svc.cluster.local"
    ...  
    ``` 

Ambassador will now be configured to output statistics to the Prometheus StatsD exporter.

**Note:** Starting with Ambassador `0.71.0`, Prometheus and scape statistics directly from Envoy's `/metrics` endpoint removing the need to configure Ambassador to output statistics to StatsD as done above. Statistics scraped from the `/metrics` endpoint are not the same as the ones scraped from StatsD. [Click here]() for a full list of statistics at each endpoint.

### Prometheus Operator

The Prometheus Operator for Kubernetes provides an easy way to manage your Prometheus deployment using Kubernetes-style resources.

Use the published YAML files to quickly deploy the Prometheus Operator in the `Monitoring` namespace.

1. Deploy the Prometheus Operator

    ```
    kubectl apply -f https://getambassador.io/yaml/monitoring/prometheus-operator.yaml
    ```

2. Deploy Prometheus

    ```
    kubectl apply -f https://getambassador.io/yaml/monitoring/prometheus.yaml
    ```

3. Create the `ServiceMonitor`

    If you are running a version higher than 0.71.0 and want to scrape metrics directly from the `/metrics` endpoint of Ambassador running in the `default` namespace:

    ```
    kubectl apply -f https://getambassador.io/yaml/monitoring/ambassador-monitor.yaml
    ```

    If you are scrapping metrics from a `statsd-sink` deployment running in the `monitoring` namespace:

    ```
    kubectl apply -f https://getambassador.io/yaml/monitoring/statsd-monitor.yaml
    ```


#### Helm




Now you will have a full deployment of Prometheus running in your cluster. 

Access the Prometheus UI to see it in action by running:
```
kubectl port-forward -n monitoring service/prometheus  9090
```
and going to http://localhost:9090/ from a web browser

In the UI, click the dropdown and see all of the stats Prometheus is able to scrape from Ambassador! 
