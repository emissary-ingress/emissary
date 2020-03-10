# Using Ambassador

Application development teams use Ambassador to manage edge policies associated with a specific service. This section of the documentation covers core Ambassador elements that are typically used by the application development team.

* [Introduction to Mappings](intro-mappings) The `Mapping` resource is the core resource used by every application development team.
* Mapping Configuration:
  * [Automatic Retries](retries)
  * [Canary Releases](canary)
  * [Circuit Breakers](circuit-breakers)
  * [Cross Origin Resource Sharing](cors)
  * HTTP Headers
    * [Header-based Routing](headers/headers)
    * [Host Header](headers/host)
    * [Adding Request Headers](headers/add_request_headers)
    * [Adding Response Headers](headers/add_response_headers)
    * [Removing Request Headers](headers/remove_request_headers)
    * [Remove Response Headers](headers/remove_response_headers)
  * [Keepalive](keepalive)
  * Protocols
    * [TCP](tcpmappings)
    * gRPC, HTTP/1.0, gRPC-Web, WebSockets
  * [RegEx-based Routing](prefix_regex)
  * [Redirects](redirects)
  * [Rewrites](rewrites)
  * [Timeouts](timeouts)
  * [Traffic Shadowing](shadowing)
* [Advanced Mapping Configuration](mappings)
* Rate Limiting
  * [Introduction to Rate Limits](rate-limits/)
  * [Rate Limiting Configuration](rate-limits/rate-limits)
* Filters and Authentication
  * [Introduction to Filters and Authentication](filters/)
  * [OAuth2 Filter](filters/oauh2)
  * [JWT Filter](filters/jwt)
  * [External Filter](filters/external)
  * [Plugin Filter](filters/plugin)
* Service Preview and Edge Control
  * [Introduction to Edge Control](edgectl/edge-control)
  * [Edge Control in CI](edgectl/edge-control-in-ci)
* [Developer Portal](dev-portal)