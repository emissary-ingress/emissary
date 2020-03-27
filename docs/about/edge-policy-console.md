# Edge Policy Console

The Ambassador Edge Stack provides you with an easy-to-use interface that so you can create custom resource definitions, download YAML files, and visit the Developer Portal, all in one place.

After you [install the Ambassador Edge Stack](../../user-guide/install), you can log in to the Edge Policy Console (EPC) to manage your deployment.

## Available Pages

The EPC makes it easy to configure what you need for a successful deployment. However, you'll also be able to use the command line to complete any of your configurations.

* [Dashboard](#dashboard)
* [Hosts](#hosts)
* [Mappings](#mappings)
* [Filters](#filters)
* [Rate Limits](#rate-limits)
* [Plugins](#plugins)
* [Resolvers](#resolvers)
* [Debugging](#debugging)
* [YAML Download](#yaml-download)
* [APIs](#apis)
* [Documentation](#documentation)
* [Support](#support)

On most pages, you have the option to click `See YAML` which will provide you the raw YAML file for your CRD. For those that want YAML changes for Git source control, all of your configuration changes will be saved to the YAML Download" tab.

You can also browse the [Edge Control](../../reference/edge-control) for information on using the `edgectl` commands for additional actions.

### Dashboard

The landing page of the EPC is your dashboard, which shows metrics for:

* counts of Hosts, Mappings, and Plugins
* System status of Envoy and Redis
* System Service health, which you can click for more details

### Hosts

Hosts are domains that are managed by Ambassador Edge Stack. On this page, you can add and manage your hosts, which configures automatic HTTPS and TLS.

See [Hosts](../../reference/host-crd) for detailed information.

### Mappings

Mappings are associations between prefix URLs and target services.

On this page, you can add new mappings and manage any existing ones. You can sort your mappings by name, namespace, and prefix.

You can also see the Envoy route table, which includes URL, service IP, and weight in regards to load balancing.

See [Mappings](../../reference/mappings) for detailed information.

### Filters

Filters allow you to configure middleware for your requests. On this page, you can add a new filter or manage an existing filter. You can sort filters by name and namespace.

See [Filters](../../reference/filter-reference) for detailed information.

### Rate Limits

Rate limits allow you to control traffic for different request classes.

On this page, you can add a new rate limit or manage existing ones. You can sort rate limits by name and namespace.

See [rate limits](../../reference/rate-limits) for more information.

### Plugins

Special plugin services enhance the functionality of Ambassador Edge Stack. These plugin services are called when Ambassador handles requests.

On this page, you can add a new plugin or manage existing plugins.

See [Plugins](../../reference/services/services) for detailed information.

### Resolvers

This page shows all of the current Resolvers that are in use to discover your services. See [Resolvers](../../reference/core/resolvers) for more information.

### Debugging

The Debugging page provides an overview of everything that is happening on your deployment of the Ambassador Edge Stack.

The **system info** box shows information such as IDs, system statuses, and other high-level details.

The **logging level** box has two buttons, `set log level to debug` and `set log level to info` which controls how verbose your logging is.

To see the logs, follow [these instructions](../../reference/debugging/#review-ambassador-logs).

The **Ambassador Configuration** box shows an immediate status along with details about the status. For example, if the status is `has issues`, it will specify some information about those issues.

The **Configuration Errors** box provides further information about any configuration errors.

 See [Debugging](../../reference/debugging) for more information.

Also take a look at the [Diagnostics](../../reference/diagnostics) resource.

### YAML Download

The YAML Download page stores all of the configuration changes you make across the EPC in one place for you to conveniently download. If you need to push files to Git, these contain the most up-to-date information.

### APIs

The APIs page shows you all of the existing APIs with documentation that you configured from the Developer Portal.

See the [Developer Portal](../../reference/dev-portal) documentation for more information.

### Documentation

The Documentation page provides you direct links to the Ambassador Edge Stack documentation (these very pages!), available resources and case studies, as well as the Ambassador blog.

### Disabling the Edge Policy Console

If necessary, you can disable external access to the Edge Policy Console using the [Ambassador module](../../reference/core/ambassador).

### Support

Need help? Check in here to ask for help on Slack, file an issue, or contact us. See [Support](../support) for additional information.
