# Ambassador and Envoy Proxy Performance

Envoy Proxy, the core L7 proxy used by Ambassador, has excellent performance and scalability. Ambassador operates as a [control plane for Envoy](/concepts/architecture), enabling Ambassador users to get raw Envoy performance. That said, there are a number of strategies that can be used to improve Ambassador and Envoy's performance.

## What do we mean by performance?

There are many metrics for performance. In this document, performance is defined as p95 latency at a given request per second (RPS) workload. While other benchmarks may focus on pure RPS, in our case, we're interested in the additional latency introduced by Ambassador at a given workload.

