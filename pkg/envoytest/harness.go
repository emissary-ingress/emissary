package envoy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datawire/dlib/dhttp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func GetLoopbackAddr(port int) string {
	return fmt.Sprintf("%s:%d", GetLoopbackIp(), port)
}

func GetLoopbackIp() string {
	_, err := exec.LookPath("envoy")
	if err == nil {
		return "127.0.0.1"
	} else {
		cmd := exec.Command("docker", "network", "inspect", "bridge", "--format={{(index .IPAM.Config 0).Gateway}}")
		buf := &bytes.Buffer{}
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			panic(errors.Wrapf(err, "error finding loopback ip"))
		}
		return strings.TrimSpace(buf.String())
	}
}

var cidCounter int64

// SetupEnvoy launches an envoy docker container that is configured to connect to the supplied ads
// address and expose the supplied portmaps. A Cleanup function is registered to shutdown the
// container at the end of the test suite.
func SetupEnvoy(t *testing.T, adsAddress string, portmaps ...string) {
	host, port, err := net.SplitHostPort(adsAddress)
	require.NoError(t, err)

	yaml := fmt.Sprintf(bootstrap, host, port)

	_, err = exec.LookPath("envoy")

	var cmd *exec.Cmd
	var cidfile string
	if err == nil {
		cmd = exec.Command("envoy", "--config-yaml", yaml)
	} else {
		counter := atomic.AddInt64(&cidCounter, 1)
		cidfile = path.Join(os.TempDir(), fmt.Sprintf("envoy-%d-%d-cid", os.Getpid(), counter))

		args := []string{"run", "--cidfile", cidfile}
		for _, pm := range portmaps {
			args = append(args, "-p", pm)
		}
		args = append(args, "--rm", "--entrypoint", "envoy", "docker.io/datawire/aes:1.6.2", "--config-yaml", yaml)
		cmd = exec.Command("docker", args...)
	}

	cmd.Stdin = os.Stdin
	var out io.Writer
	if os.Getenv("SHUTUP_ENVOY") == "" {
		out = NewPrefixer(os.Stdout, []byte("ENVOY: "))
	}
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Start()
	if err != nil {
		t.Errorf("error starting envoy: %v", err)
		return
	}

	if cidfile == "" {
		// we started envoy without a container
		t.Cleanup(func() {
			cmd.Process.Kill()
			_, err := cmd.Process.Wait()
			if err != nil {
				t.Logf("error tearing down envoy: %+v", err)
			}
		})
	} else {
		// we started envoy inside a container so we need cleanup using the container id we captured on startup
		t.Cleanup(func() {
			// try a few times just in case the test aborted super quickly
			delay := 1 * time.Second
			var cidBytes []byte
			for {
				var err error
				cidBytes, err = ioutil.ReadFile(cidfile)
				if err != nil {
					if delay < 8*time.Second {
						time.Sleep(delay)
						delay = 2 * delay
						continue
					}

					t.Logf("error reading envoy container id: %+v", err)
					return
				}
				break
			}
			defer os.Remove(cidfile)

			cid := strings.TrimSpace(string(cidBytes))

			cmd := exec.Command("docker", "kill", cid)
			err = cmd.Run()
			if err != nil {
				t.Logf("error killing envoy container %s: %+v", cid, err)
				return
			}

			cmd = exec.Command("docker", "wait", cid)
			cmd.Run()
			if err != nil {
				// No such container is an "expected" error since the container might exit before we get
				// around to waiting for it.
				if !strings.Contains(err.Error(), "No such container") {
					t.Logf("error waiting for envoy container %s: %+v", cid, err)
					return
				}
			}
		})
	}
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

// SetupRequestLogger will launch an http server that binds to the supplied addresses, responds with
// the supplied body, and records every request it receives for later examination.
func SetupRequestLogger(t *testing.T, addresses ...string) *RequestLogger {
	rl := NewRequestLogger()
	SetupServer(t, rl, addresses...)
	return rl
}

type RequestLogger struct {
	Requests []*http.Request
}

var _ http.Handler = &RequestLogger{}

func NewRequestLogger() *RequestLogger {
	return &RequestLogger{}
}

func (rl *RequestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rl.Log(r)
	w.Write([]byte("Hello World"))
}

func (rl *RequestLogger) Log(r *http.Request) {
	rl.Requests = append(rl.Requests, r)
}

// SetupServer will launch an http server that runs for the duration of the test, binds to the
// supplied addresses using the supplied handler.
func SetupServer(t *testing.T, handler http.Handler, addresses ...string) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	sc := &dhttp.ServerConfig{Handler: handler}
	for _, address := range addresses {
		// capture the value of address for the closure below
		addr := address
		wg.Add(1)
		go func() {
			err := sc.ListenAndServe(ctx, addr)
			if err != nil && err != context.Canceled {
				t.Errorf("server exited with error: %+v", err)
			}
			wg.Done()
		}()
	}
}
