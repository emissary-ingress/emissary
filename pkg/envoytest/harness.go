package envoytest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"
)

func GetLoopbackAddr(ctx context.Context, port int) (string, error) {
	ip, err := GetLoopbackIp(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", ip, port), nil
}

func GetLoopbackIp(ctx context.Context) (string, error) {
	if _, err := dexec.LookPath("envoy"); err == nil {
		return "127.0.0.1", nil
	}
	cmd := dexec.CommandContext(ctx, "docker", "network", "inspect", "bridge", "--format={{(index .IPAM.Config 0).Gateway}}")
	bs, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "error finding loopback ip")
	}
	return strings.TrimSpace(string(bs)), nil
}

var (
	cacheDevNullMu sync.Mutex
	cacheDevNull   *os.File
)

func getDevNull() (*os.File, error) {
	cacheDevNullMu.Lock()
	defer cacheDevNullMu.Unlock()
	if cacheDevNull != nil {
		return cacheDevNull, nil
	}
	var err error
	cacheDevNull, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	return cacheDevNull, err
}

// RunEnvoy runs and waits on  an envoy docker container that is configured to connect to the supplied ads
// address and expose the supplied portmaps. A Cleanup function is registered to shutdown the
// container at the end of the test suite.
func RunEnvoy(ctx context.Context, adsAddress string, portmaps ...string) error {
	var args []string
	if _, err := dexec.LookPath("envoy"); err == nil {
		args = append(args, "envoy")
	} else {
		args = append(args,
			"docker", "run",
			"--rm",
			"--interactive",
		)
		for _, pm := range portmaps {
			args = append(args,
				"--publish="+pm)
		}
		args = append(args, "--entrypoint", "envoy", "docker.io/datawire/aes:1.6.2")
	}

	host, port, err := net.SplitHostPort(adsAddress)
	if err != nil {
		return err
	}
	args = append(args, "--config-yaml", fmt.Sprintf(bootstrap, host, port))

	cmd := dexec.CommandContext(ctx, args[0], args[1:]...)
	if os.Getenv("DEV_SHUTUP_ENVOY") != "" {
		devNull, _ := getDevNull()
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	return cmd.Run()
}

// This is the bootstrap we use for starting envoy. This is hardcoded for now, but we may want to
// make it configurable for fancier tests in the future.
const bootstrap = `
{
  "node": {
    "cluster": "ambassador-default",
    "id": "test-id"
  },
  "layered_runtime": {
    "layers": [
      {
        "name": "static_layer",
        "static_layer": {
          "envoy.deprecated_features:envoy.api.v2.route.HeaderMatcher.regex_match": true,
          "envoy.deprecated_features:envoy.api.v2.route.RouteMatch.regex": true,
          "envoy.deprecated_features:envoy.config.filter.http.ext_authz.v2.ExtAuthz.use_alpha": true,
          "envoy.deprecated_features:envoy.config.trace.v2.ZipkinConfig.HTTP_JSON_V1": true,
          "envoy.reloadable_features.ext_authz_http_service_enable_case_sensitive_string_matcher": false
        }
      }
    ]
  },
  "dynamic_resources": {
    "ads_config": {
      "api_type": "GRPC",
      "grpc_services": [
        {
          "envoy_grpc": {
            "cluster_name": "ads_cluster"
          }
        }
      ]
    },
    "cds_config": {
      "ads": {}
    },
    "lds_config": {
      "ads": {}
    }
  },
  "static_resources": {
    "clusters": [
      {
        "connect_timeout": "1s",
        "dns_lookup_family": "V4_ONLY",
        "http2_protocol_options": {},
        "lb_policy": "ROUND_ROBIN",
        "load_assignment": {
          "cluster_name": "ads_cluster",
          "endpoints": [
            {
              "lb_endpoints": [
                {
                  "endpoint": {
                    "address": {
                      "socket_address": {
                        "address": "%s",
                        "port_value": %s,
                        "protocol": "TCP"
                      }
                    }
                  }
                }
              ]
            }
          ]
        },
        "name": "ads_cluster"
      }
    ]
  }
}
`

// A RequestLogger can serve HTTP on multiple ports and records all requests to .Requests for later
// examination.
type RequestLogger struct {
	Requests []*http.Request
}

func (rl *RequestLogger) Log(r *http.Request) {
	rl.Requests = append(rl.Requests, r)
}

func (rl *RequestLogger) ListenAndServeHTTP(ctx context.Context, addresses ...string) error {
	sc := &dhttp.ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rl.Log(r)
			_, _ = w.Write([]byte("Hello World"))
		}),
	}

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		ShutdownOnNonError: true,
	})

	for _, addr := range addresses {
		addr := addr // capture the value for the closure
		grp.Go(addr, func(ctx context.Context) error {
			return sc.ListenAndServe(ctx, addr)
		})
	}

	return grp.Wait()
}
