# Installing Ambassador Edge Stack

Ambassador Edge Stack can be installed in a variety of ways. The most common approach to installing Ambassador Edge Stack is directly on Kubernetes with our default, customizable manifest.

## Kubernetes


<table>
<tr>
<td>
<a href="/user-guide/getting-started"><img src="/doc-images/kubernetes.png"></a>
</td>
<td>
Ambassador Edge Stack is designed to run in Kubernetes for production. <a href="/user-guide/getting-started">Deploy to Kubernetes via YAML</a>.
</td>
</tr>
</table>

## Other methods

You can also install Ambassador Edge Stack using Helm, Docker, or Docker Compose.

<div style="border: thick solid red"> </div>
| [![Helm](/doc-images/helm.png)](/user-guide/helm) | [![Docker](/doc-images/docker.png)](/about/quickstart) | [![Docker Compose](/doc-images/docker-compose.png)](/user-guide/docker-compose)
| --- | --- | --- |
| Helm is a package manager for Kubernetes. Ambassador Edge Stack comes pre-packaged as a Helm chart. [Deploy to Kubernetes via Helm.](/user-guide/helm) | The Docker install will let you try Ambassador Edge Stack locally in seconds, but is not supported for production. [Try via Docker.](/about/quickstart) | The Docker Compose setup gives you a local development environment (a good alternative to Minikube), but is not suitable for production. [Set up with Docker Compose.](/user-guide/docker-compose)


### Ambassador Open Source

If you want to install Ambassador Open Source instead of Edge Stack, find instructions to do so <a href="/user-guide/install-ambassador-oss">here</a>.