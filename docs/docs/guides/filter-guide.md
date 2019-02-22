# Filters

Sometimes you may want Ambassador to manipulate an incoming request. Some example use cases:

* Inspect an incoming request, and add a custom header that can then be used for routing
* Add custom Authorization headers
* Validate an incoming request fits an OpenAPI specification before passing the request to a target service

Ambassador support these use cases by allowing you to execute custom logic in `Filters`. Filters are written in Golang, and managed by Ambassador Pro.

## A sample filter

In this tutorial, we'll walk through an example filter ...


...

