# Change Log

This file documents all notable changes to Ambassador Helm Chart. The release
numbering uses [semantic versioning](http://semver.org).

## v7.4.2

- Update Emissary chart image to version v2.3.2 [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.4.1

- Update Emissary chart image to version v2.3.1 [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.4.0

- Update change Emissary chart image to version v2.3.0 [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Add "lifecycle" option to main container. This can be used, for example, to add a lifecycle.preStop hook. Thanks to [Eric Totten](https://github.com/etotten) for the contribution!
- Add `ambassador_id` to listener manifests rendered when using `createDefaultListeners: true` with `AMBASSADOR_ID` set in environment variables. Thanks to [Jennifer Reed](https://github.com/ServerNinja) for the contribution!
- Feature: Added configurable IngressClass resource to be compliant with Kubernetes 1.22+ ingress specification.
  
## v7.2.2

- Update Emissary chart image to version v2.1.2: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.2.1

There was no v7.2.1 release; we skipped to v7.2.2 to keep the version
number in-sync with the edge-stack chart.

## v7.2.0

- Update Emissary chart image to version v2.1.0: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Change: It is now *always* nescessary to manually apply `emissary-crds.yaml` before applying the chart.
- Bugfix: When setting `adminService.snapshotPort`, it now points at the correct port on the Pod.

## v7.1.10

- Update Ambassador chart image to version v2.0.5: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.1.9

- Update Ambassador chart image to version v2.0.4: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.1.8-ea

- Update Ambassador chart image to version v2.0.3-ea: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.1.2-ea

- Update Ambassador chart image to version v2.0.2-ea: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.1.1-ea

- Update Ambassador chart image to version v2.0.1-ea: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v7.1.0-ea

- Feature: New canarying features for Ambassador in the chart that allow creation of a secondary deployment/service to test new versions and environment variables.
- Feature: Exposed `progressDeadlineSeconds` for the Ambassador and Ambassador Agent Deployments with new values

## v7.0.0-ea

We're pleased to introduce Emissary 2.0.0 as a developer preview.

Emissary Ingress chart v7.0.0-ea provides early access to Emissary 2.0 features. [Learn more in our docs](https://www.getambassador.io/docs/emissary/latest/about/changes-2.0.0/)

- Update Ambassador chart image to version v2.0.0-ea: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Change: removed helm 2 support
- Feature: Add an option to create default HTTP and HTTPS listeners

## v6.9.3

- Update Ambassador API Gateway chart image to version v1.14.2: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Update Ambassador Edge Stack chart image to version v1.14.2: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

(v6.9.2 was withdrawn)

## v6.9.1

- Update Ambassador API Gateway chart image to version v1.14.1: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Update Ambassador Edge Stack chart image to version v1.14.1: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v6.9.0

- Update Ambassador Edge Stack chart image to version v1.14.0: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v6.8.0

- Update Ambassador API Gateway chart image to version v1.14.0: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v6.7.14

- Feature: New Values to expose `progressDeadlineSeconds` for the Ambassador and Ambassador-agent Deployments.

## v6.7.13

- Update Ambassador API Gateway chart image to version v1.13.10: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Update Ambassador Edge Stack chart image to version v1.13.10: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v6.7.12

- Update Ambassador API Gateway chart image to version v1.13.9: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Update Ambassador Edge Stack chart image to version v1.13.9: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v6.7.11

- Update Ambassador API Gateway chart image to version v1.13.8: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Update Ambassador Edge Stack chart image to version v1.13.8: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Bugfix: remove duplicate label key in ambassador-agent deployment

## v6.7.10

- Update Ambassador API Gateway chart image to version v1.13.7: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)
- Update Ambassador Edge Stack chart image to version v1.13.7: [CHANGELOG](https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md)

## v6.7.9

- Update Ambassador chart image to version 1.13.6: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)


## v6.7.8

- Update Ambassador chart image to version 1.13.5: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.7.7

- Bugfix: ambassador-injector and telepresence-proxy now use the correct default image repository

## v6.7.6

- Update Ambassador chart image to version 1.13.4: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Change: unless image.repository or image.fullImageOverride is explicitly set, the ambassador image used will be templated on .Values.enableAES. If AES is enabled, the chart will use docker.io/datawire/aes, otherwise will use docker.io/datawire/ambassador.

## v6.7.5

- Update Ambassador chart image to version v1.13.3: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.7.4

- Feature: The [Ambassador Module](https://www.getambassador.io/docs/edge-stack/latest/topics/running/ambassador/) can now be configured and managed by Helm

## v6.7.3

- Update Ambassador chart image to version v1.13.2: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.7.2

- Bugfix: Don't change the Role name when running in singleNamespace mode.

## v6.7.1

- Update Ambassador chart image to version v1.13.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.7.0

- Update Ambassador to version 1.13.0: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Feature: Ambassador Agent now available for API Gateway (https://app.getambassador.io)
- Feature: Add support for [pod toplology spread constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/) via the `topologySpreadConstraints` helm value to the Ambassador deployment. (thanks, [@lawliet89](https://github.com/lawliet89)!)
- BugFix: Add missing `ambassador_id` for resolvers.
- Change: Ambassador ClusterRoles are now aggregated under the label `rbac.getambassador.io/role-group`. The aggregated role has the same name as the previous role name (so no need to update ClusterRoleBindings).

## v6.6.4

- Update Ambassador to version 1.12.4: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.6.3

- Update Ambassador to version 1.12.3: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.6.2

- Update Ambassador to version 1.12.2: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.6.1

- Fix metadata field in ConsulRevoler
- Make resolvers available to OSS

## v6.6.0

- Update Ambassador to version 1.12.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Feature: Apply Ambassador Agent deployment by default to enable Service Catalog reporting (https://app.getambassador.io)

## v6.5.22

- Bugfix: Disable the cloud agent by default. The agent will be enabled in 6.6.0.
- Bugfix: Adds a check to prevent the cloud agent from being installed if AES version is less than 1.12.0

## v6.5.21

- Update Ambassador to version 1.12.0: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Feature: Add support for the ambassador-agent, reporting to Service Catalog (https://app.getambassador.io)
- Feature: All services are automatically instrumented with discovery annotations.

## v6.5.20

- Update Ambassador to version v1.11.2: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.19

- Make all `livenessProbe` and `readinessProbe` configurations available to the values file

## v6.5.18

- Update Ambassador to version v1.11.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.17

- Update Ambassador to version v1.11.0: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Bugfix: Fix Mapping definition to correctly support labels in use.

## v6.5.16

- Bugfix: Ambassador CRD cleanup will now execute as expected.

## v6.5.15

- Bugfix: Ambassador RBAC now includes permissions for IngressClasses.

## v6.5.14

- Update for Ambassador v1.10.0

## v6.5.13

- Update for Ambassador v1.9.1

## v6.5.12

- Feature: Add ability to configure `terminationGracePeriodSeconds` for the Ambassador container
- Update for Ambassador v1.9.0

## v6.5.11

- Feature: add affinity and tolerations support for redis pods

## v6.5.10

- Update Ambassador to version 1.8.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.9

- Update Ambassador to version 1.8.0: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Bugfix: The RBAC for AES now grants permission to "patch" Events.v1.core.  Previously it granted "create" but not "patch".

## v6.5.8

- Update Ambassador to version 1.7.4: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.7

- Update Ambassador to version 1.7.3: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- The BusyBox image image used by `test-ready` is now configurable (thanks, [Alan Silva](https://github.com/OmegaVVeapon)!)

## v6.5.6

- Update Ambassador to version 1.7.2: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Feature: Allow overriding the namespace for the release using the values file: [ambassador-chart/#122](https://github.com/datawire/ambassador-chart/pull/122)

## v6.5.5

- Allow hyphens in service annotations: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.4

- Upgrade Ambassador to version 1.7.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.3

- Upgrade Ambassador to version 1.7.0: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.2

- Feature: Add support for DaemonSet/Deployment labels: [ambassador-chart/#114](https://github.com/datawire/ambassador-chart/pull/114)
- Upgrade Ambassador to version 1.6.2: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.1

- Upgrade Ambassador to version 1.6.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.5.0

- Upgrade Ambassador to version 1.6.0: [CHANGELOG}](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.4.10

- Feature: Allow specifying annotations for the license-key-secret: [ambassador-chart/#106](https://github.com/datawire/ambassador-chart/issues/106)
- Feature: Annotation for keeping the AES secret on removal: [ambassador-chart/#110](https://github.com/datawire/ambassador-chart/issues/110)
- Fix: do not mount the secret if we do not want a secret: [ambassador-chart/#103](https://github.com/datawire/ambassador-chart/issues/103)
- Internal CI refactorings.

## v6.4.9

- BugFix: Cannot specify podSecurityPolicies: [ambassador-chart/#97](https://github.com/datawire/ambassador-chart/issues/97)

## v6.4.8

- Upgrade Ambassador to version 1.5.5: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.4.7

- BugFix: Registry service is now using the proper `app.kubernetes.io/name`
- BugFix: Restore ability to set `REDIS` env vars in `env` instead of `redisEnv`
- Feature: Add `envRaw` to support supplying raw yaml for environment variables. Deprecates `redisEnv`.

## v6.4.6

- Upgrade Ambassador to version 1.5.4: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- Added support setting external IPs for the ambassador service (thanks, [Jason Smith](https://github.com/jasons42)!)

## v6.4.5

- Upgrade Ambassador to version 1.5.3: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.4.4

- Feature flag for enabling or disabling the [`Project` registry](https://www.getambassador.io/docs/edge-stack/latest/topics/using/projects/)
- redisEnv for setting environment variables to control how Ambassador interacts with redis. See [redis environment](https://www.getambassador.io/docs/edge-stack/latest/topics/running/environment/#redis)

## v6.4.3

- Upgrade Ambassador to version 1.5.2: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.4.2

- Upgrade Ambassador to version 1.5.1: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.4.1

- BugFix: The `PodSecurityPolicy` should not be created by default since it is a cluster-wide resource that should only be created once.

If you would like to use the default `PodSecurityPolicy`, make sure to unset `security.podSecurityPolicy` it in all other releases.

## v6.4.0

- Upgrade Ambassador to version 1.5.0: [CHANGELOG](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)
- AuthService and RateLimitService are now installed in the same namespace as Ambassador.
- Changes RBAC permissions to better support single-namespace installations and detecting getambassador.io CRDs.
- Add option to install Service Preview components (traffic-manager, traffic-agent).
- Add option to install ambassador-injector, alongside Service Preview.
- Add additional security policy configurations.

   `securityContext` has been deprecated in favor of `security` which allows you to set container and pod security contexts as well as a default `PodSecurityPolicy`.

## v6.3.6

- Switch from Quay.io to DockerHub

## v6.3.5

- Upgrade Ambassador to version 1.4.3: [CHANGELOG}](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.3.4

- Minor bug fixes

## v6.3.3

- Add extra labels to ServiceMonitor: [CHANGELOG}](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.3.2

- Upgrade Ambassador to version 1.4.2: [CHANGELOG}](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.3.1

- Upgrade Ambassador to version 1.4.1: [CHANGELOG}](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.3.0

- Adds: Option to create a ServiceMonitor for scraping via Prometheus Operator

## v6.2.5

- Upgrade Ambassador to version 1.4.0: [CHANGELOG}](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## v6.2.4

- Fix typing so that Helm3 doesn't complain (thanks, [Fabrice Rabaute](https://github.com/jfrabaute)!)

## v6.2.3

- Upgrade Ambassador to version 1.3.2.
- Use explicit types for things like ports, so that things like `helm .. --set service.ports[0].port=80` will be integers instead of ending up as strings

## v6.2.2

- Upgrade Ambassador to version 1.3.1.
- Remove unnecessary `version` field from CRDs.
- Add static label to AES resources, to better support `edgectl install`

## v6.2.1

- Upgrade Ambassador to version 1.3.0.

## v6.2.0

- Add option to not create DevPortal routes

## v6.1.5

- Upgrade Ambassador to version 1.2.2.

## v6.1.4

- Upgrade from Ambassador 1.2.0 to 1.2.1.

## v6.1.3

- Upgrade from Ambassador 1.1.1 to 1.2.0.

## v6.1.2

- Upgrade from Ambassador 1.1.0 to 1.1.1.

## v6.1.1

Minor Improvements:

- Adds: Option to override the name of the RBAC resources

## v6.1.0

Minor improvements including:

- Adds: Option to set `restartPolicy`
- Adds: Option to give the AES license key secret a custom name
- Fixes: Assumption that the AES will be installed only from the `datawire/aes` repository. The `enableAES` flag now configures whether the AES is installed.
- Clarification on how to install OSS

## v6.0.0

Introduces Ambassador Edge Stack being installed by default.

### Breaking changes

Ambassador Pro support has been removed in 6.0.0. Please upgrade to the Ambassador Edge Stack.

## v5.0.0

### Breaking changes

**Note** If upgrading an existing helm 2 installation no action is needed, previously installed CRDs will not be modified.

- Helm 3 support for CRDs was added. Specifically, the CRD templates were moved to non-templated files in the `/crds` directory, and to keep Helm 2 support they are globbed from there by `/templates/crds.yaml`. However, because Helm 3 CRDs are not templated, the labels for new installations have necessarily changed

## v4.0.0

### Breaking Changes

- Introduces the performance tuned and certified build of open source Ambassador, Ambassador core
- The license key is now stored and read from a Kubernetes secret by default
- Added `.Values.pro.licenseKey.secret.enabled` `.Values.pro.licenseKey.secret.create` fields to allow multiple releases in the same namespace to use the same license key secret.

### Minor Changes

- Introduces the ability to configure resource limits for both Ambassador Pro and it's redis instance
- Introduces the ability to configure additional `AuthService` options (see [AuthService documentation](https://www.getambassador.io/reference/services/auth-service/))
- The ambassador-pro-auth `AuthService` and ambassador-pro-ratelimit `RateLimitService` and now created as CRDs when `.Values.crds.enabled: true`
- Fixed misnamed selector for redis instance that failed in an edge case
- Exposes annotations for redis deployment and service

## v3.0.0

### Breaking Changes

- The default annotation has been removed. The service port will be set dynamically to 8080 or 8443 for http and https respectively.
- `service.http`, `service.https`, and `additionalTCPPort` has been replaced with `service.ports`.
- `rbac.namespaced` has been removed. Use `scope.singleNamespace` instead.

### Minor Changes

- Ambassador Pro will pick up when `AMBASSADOR_ID` is set in `.Values.env` [[#15025]](https://github.com/helm/charts/issues/15025).
- `{{release name}}-admins` has been renamed to `{{release name}}-admin` to match YAML install templates
- RBAC configuration has been updated to allow for CRD use when `scope.singleNamespace: true`. [[ambassador/#1576]](https://github.com/datawire/ambassador/issues/1576)
- RBAC configuration now allows for multiple Ambassadors to use CRDs. Set `crds.enabled` in releases that expect CRDs [[ambassador/#1679]](https://github.com/datawire/ambassador/issues/1679)

## v2.6.0

### Minor Changes

- Add ambassador CRDs!
- Update ambassador to 0.70.0

## v2.5.1

### Minor Changes

- Update ambassador to 0.61.1

## v2.5.0

### Minor Changes

- Add support for autoscaling using HPA, see `autoscaling` values.

## v2.4.1

### Minor Changes

- Update ambassador to 0.61.0

## v2.4.0

### Minor Changes

- Allow configuring `hostNetwork` and `dnsPolicy`

## v2.3.1

### Minor Changes

- Adds HOST_IP environment variable

## v2.3.0

### Minor Changes

- Adds support for init containers using `initContainers` and pod labels `podLabels`

## v2.2.5

### Minor Changes

- Update ambassador to 0.60.3

## v2.2.4

### Minor Changes

- Add support for Ambassador PRO [see readme](https://github.com/helm/charts/blob/master/stable/ambassador/README.md#ambassador-pro)

## v2.2.3

### Minor Changes

- Update ambassador to 0.60.2

## v2.2.2

### Minor Changes

- Update ambassador to 0.60.1

## v2.2.1

### Minor Changes

- Fix RBAC for ambassador 0.60.0

## v2.2.0

### Minor Changes

- Update ambassador to 0.60.0

## v2.1.0

### Minor Changes

- Added `scope.singleNamespace` for configuring ambassador to run in single namespace

## v2.0.2

### Minor Changes

- Update ambassador to 0.53.1

## v2.0.1

### Minor Changes

- Update ambassador to 0.52.0

## v2.0.0

### Major Changes

- Removed `ambassador.id` and `namespace.single` in favor of setting environment variables.

## v1.1.5

### Minor Changes

- Update ambassador to 0.50.3

## v1.1.4

### Minor Changes

- support targetPort specification

## v1.1.3

### Minor Changes

- Update ambassador to 0.50.2

## v1.1.2

### Minor Changes

- Add additional chart maintainer

## v1.1.1

### Minor Changes

- Default replicas -> 3

## v1.1.0

### Minor Changes

- Allow RBAC to be namespaced (`rbac.namespaced`)

## v1.0.0

### Major Changes

- First release of Ambassador Helm Chart in helm/charts
- For migration see [Migrating from datawire/ambassador chart](https://github.com/helm/charts/tree/master/stable/ambassador#migrating-from-datawireambassador-chart-chart-version-0400-or-0500)
