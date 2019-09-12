# Dev Portal

## Rendering API Documentation

The Dev Portal will automatically discover all services known by Ambassador (i.e., have a valid `Mapping`). For each `prefix` in a `Mapping`, the Dev Portal will attempt to fetch a Swagger or OpenAPI specification from `$PREFIX/.ambassador-internal/openapi-docs/`. You will need to update your microservice to return a Swagger or OAPI document at this URL.

### `.ambassador-internal`

By default, `.ambassador-internal` is not publicly exposed by Ambassador. This is controlled by a special `FilterPolicy` called `apro-internal-access-control`.

 Note that these URLs are not publicly exposed by Ambassador, and are internal-only.

### Polling interval

The Dev Portal periodically polls all mappings for updated Swagger specification. You can adjust the interval for polling with the `POLL_EVERY_SECS` environment variable.

## Styling the Dev Portal

The look and feel of the Dev Portal can be fully customized for your particular organization. In addition, additional content on your API documentation (e.g., best practices, usage tips, etc.) can be easily added.

The default Dev Portal styles are hosted in GitHub: https://github.com/datawire/devportal-content. To use your own styling, clone or copy the repository, and update the `APRO_DEVPORTAL_CONTENT_URL` environment variable to point to the repository. If you wish to use a private GitHub repository, create a [personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) and include the PAT in the `APRO_DEVPORTAL_CONTENT_URL` variable following the example below:

https://9cb034008ddfs819da268d9z13b7ecd26@github.com/datawire/private-repo