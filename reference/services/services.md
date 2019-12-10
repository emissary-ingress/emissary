# Available Plugins

You may need an API Gateway to enforce policies specific to your organization. Ambassador Edge Stack supports custom policies through external service plugins. The policy logic specific to your organization is implemented in the external service, and Ambassador is configured to send RPC requests to your service.

Currently, Ambassador Edge Stack supports plugins for authentication, rate limiting and tracing.

* [AuthService](/reference/services/auth-service) Plugin
* [RateLimitService](/reference/services/rate-limit-service) Plugin
* [TracingService](/reference/services/tracing-service) Plugin
* [FilterPolicy](/reference/services/access-control)
