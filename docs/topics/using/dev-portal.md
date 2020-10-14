# Developer Portal

## Rendering API Documentation

The _Dev Portal_ uses the `Mapping` resource to automatically discover services know by
the Ambassador Edge Stack.

For each `Mapping`, the _Dev Portal_ will attempt to fetch an OpenAPI V3 document from the
upstream service.

### `docs` settings in Mappings

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
fetching the documentation. You will need to update your microservice to return
a Swagger or OAPI document at this URL.
* `url`:  absolute URL to a OpenAPI V3 document.
* `ignored`: ignore this Mapping for documenting services. Note that the service
will appear in the Dev Portal anyway if another, non-ignored `Mapping` exists.

> Note:
>
> Previous versions of the _Dev Portal_ tried to obtain documentation from
> `/.ambassador-internal/openapi` by default. Users can set `docs.path` to
> `/.ambassador-internal/openapi` in their `Mapping`s for keeping this
> behavior.

Example:

With the `Mapping`s below, the _Dev Portal_ would fetch OpenAPI documentation
from `service-a:5000` at the path `/srv/openapi/` and from `httpbin` from an
external URL. `service-b` will have no documentation.

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
    path: /openapi/
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  service-b
spec:
  prefix: /service-b/
  service: service-b
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
> By default, all the `path`s where documentation has been found will NOT be publicly
> exposed by the Ambassador Edge Stack. This is controlled by a special
> `FilterPolicy` installed internally.

### Publishing the documentation

All rendered API documentation is published at the `/docs/` URL by default. However,
users can achieve a higher level of customization by creating
`DevPortal` resources. These resources allow the customization of:

- _what_ documentation is published
- _how_ it looks

defined with the following syntax:

```yaml
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  "string"
spec:
  content:
    url: "string"         ## optional; see below
    branch: "string"      ## optional; see below
    dir: "string"         ## optional; see below
  selector:
    matchnamespaces:      ## optional; default is []
      - "string"
    matchLabels:          ## optional; default is {}
      "string": "string"
```

where:

* `content`: see [section below](#styling).
* `matchNamespaces`: list of namespaces, used for filtering the `Mapping`s that
will be shown in the `DevPortal`. When multiple namespaces are provided, the `DevPortal`
will consider `Mapping`s in **any** of those namespaces.
* `matchLabels`: dictionary of labels, filtering the `Mapping`s that will
be shown in the `DevPortal`. When multiple labels are provided, the `DevPortal`
will consider `Mapping`s that match **all** the labels.

Example:

The scope of the default documentation can be restricted to
`Mappings` with the `private-api: true` label by creating a `DevPortal` resource
like this:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  dev-portal-private
spec:
  content:
    url: https://github.com/datawire/devportal-content
  selector:
    matchLabels:
      private-api: true    ## label for matching only some Mappings
```

Then we must edit the current `ambassador-devportal` `Mapping` with

```console
kubectl get mappings -n ambassador ambassador-devportal -o yaml
```

and specify the `dev-portal-external` in the `rewrite`:

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: ambassador-devportal
spec:
  prefix: /docs/
  rewrite: /docs/dev-portal-external   ## name of the DevPortal resource
  service: 127.0.0.1:8500
```

#### Customizing DevPortals for different audiences

The same documentation could be published in a different _host_ and
with a different _content_ with a second `DevPortal` and `Mapping`,
with something like:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  dev-portal-quickstart
spec:
  content:
    url: https://git-repo.intranet/docserver/quickstart  ## customized contents
  selector:
    matchLabels:
      public-api: true              ## matches only public-api `Mapping`s
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  dev-portal-quickstart
spec:
  host: public.my-company.com       ## match public.my-company.com/quickstart/
  prefix: /quickstart/
  rewrite: "/docs/dev-portal-quickstart"
  service: localhost:8500
```

This would show the documentation for all the `public-api` Mapings at
`public.my-company.com/quickstart/`, rendered with the `quickstart` templates located
in `https://git-repo.intranet/docserver/quickstart`.

You can create as many `DevPortal`s as you want as long as the `rewrite` uses the
`/docs/<devportal-name>`. For example, intranet users could access all the
documentation (both public and private) at `docserver.intranet/apis/` with:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  dev-portal-intranet
spec:
  content:
    url: https://git-repo.intranet/docserver/content
  selector:
    matchLabels:
      public-api: true     ## will match any `Mappings` with the `public-api`
      private-api: true    ## >... AND `private-api` labels
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  dev-portal-intranet
spec:
  host: docserver.intranet               ## an internal host for publishing docs
  prefix: /apis/
  rewrite: "/docs/dev-portal-intranet"   ## ref to the dev-portal-quickstart DevPortal
  service: localhost:8500
```

> Note that intranet users could still access the "public view" of your docs at
> `public.my-company.com/quickstart/`, unless prohibitted by the network policies in
> the cluster.

#### Adding authentication

You could also add authentication for accessing your documentation by leveraging
the Ambassador Edge Stack
[filters and filter policies](https://www.getambassador.io/docs/latest/topics/using/filters/).
For example, you can protect with _OAuth2_ the intranet docs we previously published at
`docserver.intranet/apis/`  with something like:

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: auth-filter
  namespace: default
spec:
  OAuth2:
    authorizationURL: PROVIDER_URL   ## the URL of the OAuth2 descriptor
    clientID: CLIENT_ID              ## OAuth2 client from your IdP
    secret: CLIENT_SECRET            ## Secret used to access OAuth2 client
    protectedOrigins:
    - origin: docserver.intranet
---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: intranet-docs-policy
spec:
  rules:
  - host: docserver.intranet
    path: /apis
    filters:
    - name: auth-filter
```

#### <a href="#styling"></a>Styling the `DevPortal`

The look and feel of a `DevPortal` can be fully customized for your particular
organization by specifying a different `content`, customizing not only _what_
is shown but _how_ it is shown, and giving the possibility to
add some specific content on your API documentation (e.g., best practices,
usage tips, etc.) depending on where it has been published.

The default _Dev Portal_ content is loaded in order from:

- the Git repo specified in the optional `DEVPORTAL_CONTENT_URL` environment variable
- the default repository at [GitHub](https://github.com/datawire/devportal-content.git).

To use your own styling, clone or copy the repository, and update the
`content` attribute to point to the repository. If you wish to use a private GitHub
repository, create a [personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line)
and include the _PAT_ in the `content` following the example below:

```yaml
---
apiVersion: getambassador.io/v2
kind:  DevPortal
metadata:
  name:  dev-portal
spec:
  host: mycompany.com
  path: /external-docs
  content:
    url: https://9cb034008ddfs819da268d9z13b7ecd26@github.com/datawire/private-devportal-repo
  selector:
    matchLabels:
      public-api: true
```

The `content` can be have the following attributes:

```yaml
  content:
    url: "string"      ## optional; defaults is the default repo
    branch: "string"   ## optional; defaults is "master"
    dir: "string"      ## optional; defaults is  "/"
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
devportal content will be reflected immediately on page refresh

## <a href="#global-config"></a>Default Configuration

The _Dev Portal_ supports some default configuration in the `devportal` section in the
[ambassador `Module`](https://www.getambassador.io/docs/latest/topics/running/ambassador/)
as well as with some environment variables (for backwards compatibility).

### `ambassador` `Module`

The same configuration can be provided in the `ambassador` `Module`:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
    devportal:
      poll: integer      ## optional; default is 60
      content:
        url: "string"    ## optional; default is ""
        dir: "string"    ## optional; default is ""
        branch: "string" ## optional; default is ""
```

where

* `poll`: default poll interval (in seconds)
* `content.url`: default URL to the repository hosting the content for the Portal
* `content.dir`: default content subdir
* `content.branch`: default content branch


### Environment variables

The _Dev Portal_ can also obtain the default configuration from environment variables
defined in the AES `Deployment`. This configuration method is considered deprecated and
kept only for backwards compatibility: users should configure the default values with
the `ambassador` Module.

| Setting                          |   Description       |
| -------------------------------- | ------------------- |
| AMBASSADOR_URL                   | External URL of Ambassador Edge Stack; include the protocol (e.g., `https://`) |
| POLL_EVERY_SECS                  | Interval for polling OpenAPI docs; default 60 seconds |
| DEVPORTAL_CONTENT_URL            | Default URL to the repository hosting the content for the Portal |
| DEVPORTAL_CONTENT_DIR            | Deafult content subdir (defaults to `/`) |
| DEVPORTAL_CONTENT_BRANCH         | Default content branch (defaults to `master`) |
