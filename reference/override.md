# Customizing Envoy

You may run into a situation where Ambassador may not yet support a specific Envoy feature. In these situations, please file a[GitHub issue](https://github.com/datawire/ambassador/issues/) and/or contact us on [Slack](https://d6e.co/slack) with the specific feature request.

If you're able to contribute, a fully rendered `envoy.json` file with the appropriate Envoy configuration that you want to enable will accelerate the development process.

## `envoy_override` etc

In Ambassador 0.40.2 and earlier, a custom Jinja2 template was used to render the Envoy configuration. This approach worked well with Envoy's v1 configuration format. After Ambassador 0.40.2, the Jinja2 approach is not used. Instead, an in-memory intermediate representation (IR) of your configuration is created, and Envoy configuration is generated directly from the IR. This approach has two major benefits: more sophisticated validation and the ability to support Envoy's v2 configuration. Unfortunately, this means that the previously supported method of customizing Jinja2 templates does not work. We hope to introduce a better `envoy_override` mechanism in future versions of Ambassador that fully supports v2 configuration.