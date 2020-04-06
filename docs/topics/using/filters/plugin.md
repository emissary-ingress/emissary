# Filter Type: `Plugin`

The `Plugin` filter type allows you to plug in your own custom code. This code is compiled to a `.so` file, which you load into the Ambassador Edge Stack container at `/etc/ambassador-plugins/${NAME}.so`. For a tutorial on developing filters, see the [Filter Development Guide](../../../../howtos/filter-dev-guide).

## The Plugin Interface

This code is written in the Go programming language (Golang), and must be compiled with the exact same compiler settings as the Ambassador Edge Stack; and any overlapping libraries used must have their versions match exactly. This information is documented in the `/ambassador/aes-abi.txt` file in the AES docker image.

Plugins are compiled with `go build -buildmode=plugin -trimpath`, and must have a `main.PluginMain` function with the signature `PluginMain(w http.ResponseWriter, r *http.Request)`:

```go
package main

import (
	"net/http"
)

func PluginMain(w http.ResponseWriter, r *http.Request) { … }
```

`*http.Request` is the incoming HTTP request that can be mutated or intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.
Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200 (`http.StatusOK`) instead of modifying the request, the plugin has
taken over the request, and the request will not be sent to your backend service.  You can call `w.Write()` to write the body of an error page.

## `Plugin` Global Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-plugin-filter"
  namespace: "example-namespace"
spec:
  Plugin:
    name: "string" # required; this tells it where to look for the compiled plugin file; "/etc/ambassador-plugins/${NAME}.so"
```

## `Plugin` Path-Specific Arguments

Path specific arguments are not supported for Plugin filters at this time.