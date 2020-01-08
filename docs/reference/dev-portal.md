# Developer Portal

## Rendering API Documentation

The Dev Portal will automatically discover all services known by the Ambassador Edge Stack (i.e., have a valid `Mapping`). For each `prefix` in a `Mapping`, the Dev Portal will attempt to fetch a Swagger or OpenAPI specification from `$PREFIX/.ambassador-internal/openapi-docs`. You will need to update your microservice to return a Swagger or OAPI document at this URL.

### `/docs/`

Rendered API documentation is published at the `/docs/` URL by default. In a subsequent release, support will be added for publish at alternative URLs.

### `.ambassador-internal`

By default, `.ambassador-internal` is not publicly exposed by the Ambassador Edge Stack. This is controlled by a special `FilterPolicy` called `ambassador-internal-access-control`.

 Note that these URLs are not publicly exposed by the Ambassador Edge Stack, and are internal-only.

## Dev Portal configuration

The Dev Portal supports configuring the following environment variables for configuration:

| Setting                          |   Description       |
| -------------------------------- | ------------------- |
| AMBASSADOR_URL                   | External URL of Ambassador Edge Stack; include the protocol (e.g., `https://`) |
| DEVPORTAL_CONTENT_URL            | URL to the repository hosting the content for the Portal |
| POLL_EVERY_SECS                  | Interval for polling OpenAPI docs; default 60 seconds |
| DEVPORTAL_CONTENT_DIR            | Defaults to `/` |
| DEVPORTAL_CONTENT_BRANCH         | Defaults to `master` |

## Styling the Dev Portal

The look and feel of the Dev Portal can be fully customized for your particular organization. In addition, additional content on your API documentation (e.g., best practices, usage tips, etc.) can be easily added.

The default Dev Portal styles are hosted in [GitHub](https://github.com/datawire/devportal-content.git). To use your own styling, clone or copy the repository, and update the `DEVPORTAL_CONTENT_URL` environment variable to point to the repository. If you wish to use a private GitHub repository, create a [personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) and include the PAT in the `DEVPORTAL_CONTENT_URL` variable following the example below:

```
https://9cb034008ddfs819da268d9z13b7ecd26@github.com/datawire/private-devportal-repo
```

### Iterating on Dev Portal styling and content

Check out a local copy of your content repo (see `DEVPORTAL_CONTENT_URL` above) and from within run the following docker image:

```
docker run -it --rm --volume $PWD:/content --publish 8877:8877 quay.io/datawire/ambassador_pro:local-devportal-$aproVersion$
```

and open `http://localhost:8877` in your browser. Any changes made locally to devportal content will be reflected immediately on page refresh

## Customizing the Dev Portal URL prefix

Default Dev Portal prefix is `/docs/`. To change the prefix, edit the `ambassador` Mapping CRD named `ambassador-devportal`. Change the `prefix` to your desired prefix (for example `/documentation/`) and change the `rewrite` to `/docs/`

Note: Dev portal uses another mapping named `ambassador-devportal-api` which, for now should not be changed. This restriction will be removed in a future release.
