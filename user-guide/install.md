# Installing Ambassador

Ambassador can be installed in a variety of ways. The most common approach to installing Ambassador is directly on Kubernetes with our default, customizable manifest.

## Kubernetes


<table>
<tr>
<td>
<a href="/user-guide/getting-started"><img src="/doc-images/kubernetes.png"></a>
</td>
<td>
Ambassador is designed to run in Kubernetes for production. <a href="/user-guide/getting-started">Deploy to Kubernetes via YAML</a>.
</td>
</tr>
</table>

## Other methods

You can also install Ambassador using Helm, Docker, or Docker Compose.

| [![Helm](/doc-images/helm.png)](/user-guide/helm) | [![Docker](/doc-images/docker.png)](/about/quickstart) | [![Docker Compose](/doc-images/docker-compose.png)](/user-guide/docker-compose)
| --- | --- | --- |
| Helm is a package manager for Kubernetes. Ambassador comes pre-packaged as a Helm chart. [Deploy to Kubernetes via Helm.](/user-guide/helm) | The Docker install will let you try Ambassador locally in seconds, but is not supported for production. [Try via Docker.](/about/quickstart) | The Docker Compose setup gives you a local development environment (a good alternative to Minikube), but is not suitable for production. [Set up with Docker Compose.](/user-guide/docker-compose)