# Developer Portal

## Rendering API Documentation

The _Dev Portal_ uses the `Mapping` resource to automatically discover services known by
the Ambassador Edge Stack.

For each `Mapping`, the _Dev Portal_ will attempt to fetch an OpenAPI V3 document
when a `docs` attribute is specified.

### `docs` attribute in Mappings

This documentation endpoint is defined by the optional `docs` attribute in the `Mapping`.

```yaml
  docs:
    path: "string"   # optional; default is ""
    url: "string"    # optional; default is ""
    ignored: bool    # optional; default is false
```

where:

* `path`: path for the OpenAPI V3 document.
The Ambassador Edge Stack will append the value of `docs.path` to the `prefix`
in the `Mapping` so it will be able to use Envoy's routing capabilities for
fetching the documentation from the upstream service . You will need to update
your microservice to return a Swagger or OAPI document at this URL.
* `url`:  absolute URL to an OpenAPI V3 document.
* `ignored`: ignore this `Mapping` for documenting services. Note that the service
will appear in the _Dev Portal_ anyway if another, non-ignored `Mapping` exists
for the same service.

> Note:
>
> Previous versions of the _Dev Portal_ tried to obtain documentation automatically
> from `/.ambassador-internal/openapi-docs` by default, while the current version
> will not try to obtain documentation unless a `docs` attribute is specified.
> Users should set `docs.path` to `/.ambassador-internal/openapi-docs` in their `Mapping`s
> in order to keep the previous behavior.

Example:

With the `Mapping`s below, the _Dev Portal_ would fetch OpenAPI documentation
from `service-a:5000` at the path `/srv/openapi/` and from `httpbin` from an
external URL. `service-b` would have no documentation.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  service-a
spec:
  prefix: /service-a/
  rewrite: /srv/
  service: service-a:5000
  docs:
    path: /openapi/            ## docs will be obtained from `/srv/openapi/`
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  service-b
spec:
  prefix: /service-b/
  service: service-b           ## no `docs` attribute, so service-b will not be documented
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: regular-httpbin
spec:
  host_rewrite: httpbin.org
  prefix: /httpbin/
  service: httpbin.org
  docs:
    url: https://api.swaggerhub.com/apis/helen/httpbin/1.0-oas3/swagger.json
```

> Notes on access to documentation `path`s:
>
> By default, all the `path`s where documentation has been found will **NOT** be publicly
> exposed by the Ambassador Edge Stack. This is controlled by a special
> `FilterPolicy` installed internally.

> Limitations on Mappings with a `host` attribute
>
> The Dev Portal will ignore `Mapping`s that contain `host`s that cannot be
> parsed as a valid hostname, or use a regular expression (when `host_regex: true`).

### Publishing the documentation

All rendered API documentation is published at the `/docs/` URL by default. Users can
achieve a higher level of customization by creating a `DevPortal` resource.
`DevPortal` resources allow the customization of:

- _what_ documentation is published
- _how_ it looks

Users can create a `DevPortal` resource for specifying the default configuration for
the _Dev Portal_, filtering `Mappings` and namespaces and specifying the content.

> Note: when several `DevPortal` resources exist, the Dev Portal will pick a random
> one and ignore the rest. A specific `DevPortal` can be used as the default configuration
> by setting the `default` attribute to `true`. Future versions will
> use other `DevPortals` for configuring alternative _views_ of the Dev Portal.

`DevPortal` resources have the following syntax:

```yaml
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  "string"
  namespace: "string"
spec:
  default: bool           ## optional; default false
  docs:                   ## optional; default is []
    - service: "string"   ## required
      url: "string"       ## required
  content:                ## optional
    url: "string"         ## optional; see below
    branch: "string"      ## optional; see below
    dir: "string"         ## optional; see below
  selector:               ## optional
    matchNamespaces:      ## optional; default is []
      - "string"
    matchLabels:          ## optional; default is {}
      "string": "string"
```

where:

* `default`: `true` when this is the default Dev Portal configuration.
* `content`: see [section below](#styling).
* `selector`: rules for filtering `Mapping`s:
  * `matchNamespaces`: list of namespaces, used for filtering the `Mapping`s that
  will be shown in the `DevPortal`. When multiple namespaces are provided, the `DevPortal`
  will consider `Mapping`s in **any** of those namespaces.
  * `matchLabels`: dictionary of labels, filtering the `Mapping`s that will
  be shown in the `DevPortal`. When multiple labels are provided, the `DevPortal`
  will onbly consider the `Mapping`s that match **all** the labels.
* `docs`: static list of _service_/_documentation_ pairs that will be shown
  in the _Dev Portal_. Only the documentation from this list will be shown in the _Dev Portal_
  (unless additional docs are included with a `selector`).
  * `service`: service name used for listing user-provided documentation.
  * `url`: a full URL to a OpenAPI document for this service. This document will be
  served _as it is_, with no extra processing from the _Dev Portal_ (besides replacing
  the _hostname_).

Example:

The scope of the default _Dev Portal_ can be restricted to
`Mappings` with the `public-api: true` and `documented: true` labels by creating
a `DevPortal` `ambassador` resource like this:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  ambassador
spec:
  default: true
  content:
    url: https://github.com/datawire/devportal-content.git
  selector:
    matchLabels:
      public-api: "true"    ## labels for matching only some Mappings
      documented: "true"    ## (note that "true" must be quoted)
```

Example:

The _Dev Portal_ can show a static list OpenAPI docs. In this example, a `eks.aws-demo`
_service_ is shown with the documentation obtained from a URL. In addition,
the _Dev Portal_ will show documentation for all the services discovered in the
`aws-demo` namespace:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  ambassador
spec:
  default: true
  docs:
    - service: eks.aws-demo
      url: https://api.swaggerhub.com/apis/kkrlogistics/amazon-elastic_kubernetes_service/2017-11-01/swagger.json
  selector:
    matchNamespaces:
      - aws-demo            ## matches all the services in the `aws-demo` namespace
                            ## (note that Mappings must contain a `docs` attribute)
```

#### <a href="#styling"></a>Styling the `DevPortal`

The look and feel of a `DevPortal` can be fully customized for your particular
organization by specifying a different `content`, customizing not only _what_
is shown but _how_ it is shown, and giving the possibility to
add some specific content on your API documentation (e.g., best practices,
usage tips, etc.) depending on where it has been published.

The default _Dev Portal_ content is loaded in order from:

- the `ambassador` `DevPortal` resource.
- the Git repo specified in the optional `DEVPORTAL_CONTENT_URL` environment variable.
- the default repository at [GitHub](https://github.com/datawire/devportal-content.git).

To use your own styling, clone or copy the repository, create an `ambassado` `DevPortal`
and update the `content` attribute to point to the repository. If you wish to use a
private GitHub repository, create a [Personal Access Token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line)
and include it in the `content` following the example below:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  ambassador
spec:
  default: true
  content:
    url: https://9cb034008ddfs819da268d9z13b7ecd26@github.com/datawire/private-devportal-repo.git
  selector:
    matchLabels:
      public-api: true
```

The `content` can be have the following attributes:

```yaml
  content:
    url: "string"      ## optional; default is the default repo
    branch: "string"   ## optional; default is "master"
    dir: "string"      ## optional; default is  "/"
```

where:

* `url`: Git URL for the content
* `branch`: the Git branch
* `dir`: subdirectory in the Git repo

#### Iterating on _Dev Portal_ styling and content

Check out a local copy of your content repo and from within run the following docker image:

```command-line
docker run -it --rm --volume $PWD:/content --publish 8877:8877 \
  docker.io/datawire/ambassador_pro:local-devportal-$aproVersion$
```

and open `http://localhost:8877` in your browser. Any changes made locally to
devportal content will be reflected immediately on page refresh.

## <a href="#global-config"></a>Default Configuration

The _Dev Portal_ supports some default configuration in some environment variables
(for backwards compatibility).

### Environment variables

The _Dev Portal_ can also obtain some default configuration from environment variables
defined in the AES `Deployment`. This configuration method is considered deprecated and
kept only for backwards compatibility: users should configure the default values with
the `ambassador` `DevPortal`.

| Setting                          |   Description       |
| -------------------------------- | ------------------- |
| AMBASSADOR_URL                   | External URL of Ambassador Edge Stack; include the protocol (e.g., `https://`) |
| POLL_EVERY_SECS                  | Interval for polling OpenAPI docs; default 60 seconds |
| DEVPORTAL_CONTENT_URL            | Default URL to the repository hosting the content for the Portal |
| DEVPORTAL_CONTENT_DIR            | Deafult content subdir (defaults to `/`) |
| DEVPORTAL_CONTENT_BRANCH         | Default content branch (defaults to `master`) |
