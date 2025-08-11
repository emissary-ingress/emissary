package envoytest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"

	"gopkg.in/yaml.v2"
)

func GetLoopbackAddr(ctx context.Context, port int) (string, error) {
	ip, err := GetLoopbackIp(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", ip, port), nil
}

func GetLoopbackIp(ctx context.Context) (string, error) {
	cmd := dexec.CommandContext(ctx, "docker", "network", "inspect", "bridge", "--format={{(index .IPAM.Config 0).Gateway}}")
	bs, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "error finding loopback ip")
	}
	return strings.TrimSpace(string(bs)), nil
}

func getOSSHome(ctx context.Context) (string, error) {
	dat, err := dexec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(dat)), nil
}

func getLocalEnvoyImage(ctx context.Context) (string, error) {
	if env := os.Getenv("ENVOY_DOCKER_TAG"); env != "" { // Same env-var as tests/utils.py:assert_valid_envoy_config()
		return env, nil
	}

	// Use the Envoy image listed in gorel.prologue.
	ossHome, err := getOSSHome(ctx)
	if err != nil {
		return "", err
	}

	goreleaserConfigPath := filepath.Join(ossHome, "gorel.prologue")
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		return "", err
	}

	var config struct {
		Env []string `yaml:"env"`
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return "", err
	}

	for _, env := range config.Env {
		if strings.HasPrefix(env, "ENVOY_IMAGE=") {
			return strings.TrimPrefix(env, "ENVOY_IMAGE="), nil
		}
	}

	return "", errors.New("no ENVOY_IMAGE found in .goreleaser.yaml")
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

func LocalEnvoyCmd(ctx context.Context, dockerFlags, envoyFlags []string) (*dexec.Cmd, error) {
	image, err := getLocalEnvoyImage(ctx)
	if err != nil {
		return nil, err
	}

	cmdline := []string{"docker", "run", "--rm"}
	cmdline = append(cmdline, dockerFlags...)
	cmdline = append(cmdline, image)
	cmdline = append(cmdline, envoyFlags...)

	cmd := dexec.CommandContext(ctx, cmdline[0], cmdline[1:]...)
	if os.Getenv("DEV_SHUTUP_ENVOY") != "" {
		devNull, _ := getDevNull()
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	return cmd, nil
}

// RunEnvoy runs and waits on  an envoy docker container that is configured to connect to the supplied ads
// address and expose the supplied portmaps. A Cleanup function is registered to shutdown the
// container at the end of the test suite.
func RunEnvoy(ctx context.Context, adsAddress string, portmaps ...string) error {
	dockerFlags := []string{
		"--interactive",
	}
	for _, pm := range portmaps {
		dockerFlags = append(dockerFlags,
			"--publish="+pm)
	}

	host, port, err := net.SplitHostPort(adsAddress)
	if err != nil {
		return err
	}
	envoyFlags := []string{
		"--config-yaml", fmt.Sprintf(bootstrap, host, port),
	}

	cmd, err := LocalEnvoyCmd(ctx, dockerFlags, envoyFlags)
	if err != nil {
		return err
	}

	return cmd.Run()
}

// TODO(lance) - this makes the test brittle and breaks when we change bootstrap configuration
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
          "envoy.reloadable_features.no_extension_lookup_by_name": false,
          "re2.max_program_size.error_level": 200
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
      ],
		  "transport_api_version": "V3"
    },
    "cds_config": {
      "ads": {},
		  "resource_api_version": "V3"
    },
    "lds_config": {
      "ads": {},
		  "resource_api_version": "V3"
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
