package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/util"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/kubeapply"
	"github.com/datawire/ambassador/pkg/tpu"

	"github.com/datawire/apro/lib/licensekeys"
)

var traffic = &cobra.Command{
	Use:   "traffic [subcommand]",
	Short: "Manage traffic in your cluster",
}

func init() {
	apictl.AddCommand(traffic)
}

var initialize = &cobra.Command{
	Use:   "initialize",
	Short: "Initialize the traffic management subsystem",
	RunE:  doInitialize,
}

func init() {
	traffic.AddCommand(initialize)
}

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

var TRAFFIC_MANAGER = template.Must(template.New("TRAFFIC_MANAGER").Parse(`
---
apiVersion: v1
kind: Service
metadata:
  name: telepresence-proxy
spec:
  type: ClusterIP
  clusterIP: None
  selector:
    app: telepresence-proxy
  ports:
  - name: sshd
    protocol: TCP
    port: 8022
  - name: api
    protocol: TCP
    port: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telepresence-proxy
  labels:
    app: telepresence-proxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: telepresence-proxy
  template:
    metadata:
      labels:
        app: telepresence-proxy
    spec:
      containers:
      - name: telepresence-proxy
        image: {{.PROXY_IMAGE}}
        command: ["traffic-manager"]
        ports:
        - name: sshd
          containerPort: 8022
        env:
        - name: AMBASSADOR_LICENSE_KEY
          value: {{.AMBASSADOR_LICENSE_KEY}}
`))

func doInitialize(cmd *cobra.Command, args []string) error {
	if err := licenseClaims.RequireFeature(licensekeys.FeatureTraffic); err != nil {
		return err
	}

	info := k8s.NewKubeInfo("", "", "")

	license_key, _ := cmd.Flags().GetString("license-key")

	input := &strings.Builder{}
	err := TRAFFIC_MANAGER.Execute(input, map[string]string{
		"PROXY_IMAGE":            getenvDefault("PROXY_IMAGE", "quay.io/datawire/aes:"+Version),
		"AMBASSADOR_LICENSE_KEY": license_key,
	})
	if err != nil {
		return err
	}

	kubectl, err := info.GetKubectl("apply -f -")
	if err != nil {
		return err
	}
	apply := tpu.NewKeeper("KAP", "kubectl "+kubectl)
	apply.Input = input.String()
	apply.Limit = 1
	apply.Start()
	apply.Wait()

	/*
		// Commenting this out because Watcher no longer works this way and
		// KubeApply seems to require having files on the filesystem.

		client, err := k8s.NewClient(info)
		if err != nil {
			return err
		}
		w, err := kubeapply.NewWaiter(client.Watcher())
		if err != nil {
			return err
		}
		err = w.Add(fmt.Sprintf("service/telepresence-proxy.%s", info.Namespace))
		if err != nil {
			return err
		}
		err = w.Add(fmt.Sprintf("deployment/telepresence-proxy.%s", info.Namespace))
		if err != nil {
			return err
		}
		if !w.Wait(time.Now().Add(30 * time.Second)) {
			return errors.New("Telepresence-proxy did not come up. Investigate: kubectl get all -l app=telepresence-proxy")
		}
	*/

	fmt.Println("Traffic management subsystem initialized. Examine using:")
	fmt.Println("  kubectl get all -l app=telepresence-proxy")
	return nil
}

var inject = &cobra.Command{
	Use:     "inject [flags] <manifest files...>",
	Short:   "Inject the traffic sidecar into a deployment",
	Args:    cobra.MinimumNArgs(1),
	Example: "apictl traffic inject -d example-dep -s example-svc -p 9000 k8s/example.yaml > injected-example.yaml",
	RunE:    doInject,
}

func init() {
	traffic.AddCommand(inject)
	inject.Flags().StringVarP(&deployment, "deployment", "d", "", "deployment to modify")
	inject.Flags().StringVarP(&service, "service", "s", "", "service to modify")
	inject.Flags().IntVarP(&port, "port", "p", 0, "application port")
	inject.MarkFlagRequired("deployment")
	inject.MarkFlagRequired("port")
}

var deployment string
var service string
var port int

func doInject(cmd *cobra.Command, args []string) error {
	if err := licenseClaims.RequireFeature(licensekeys.FeatureTraffic); err != nil {
		return err
	}

	for _, arg := range args {
		resources, err := kubeapply.LoadResources(arg)
		if err != nil {
			return err
		}
		for _, res := range resources {
			if strings.ToLower(res.Kind()) == "deployment" && res.Name() == deployment {
				err = munge(res)
				if err != nil {
					return err
				}
			}
			if strings.ToLower(res.Kind()) == "service" && res.Name() == service {
				err = mungeService(res)
				if err != nil {
					return err
				}
			}
		}
		bytes, err := kubeapply.MarshalResources(resources)
		if err != nil {
			return err
		}
		fmt.Print(string(bytes))
	}
	return nil
}

func typecastList(in interface{}) []interface{} {
	if in == nil {
		return nil
	}
	return in.([]interface{})
}

func munge(res k8s.Resource) error {
	podSpec := res.Spec()["template"].(map[string]interface{})["spec"].(map[string]interface{})

	var app_port string
	if port == 0 {
		// inspect the current list of containers to infer the app_port
		var ports []string
		for _, c := range typecastList(podSpec["containers"]) {
			iportSpecs, ok := c.(map[string]interface{})["ports"]
			if !ok {
				continue
			}

			portSpecs := typecastList(iportSpecs)

			for _, portSpec := range portSpecs {
				p, ok := portSpec.(map[string]interface{})["containerPort"]
				if ok {
					ports = append(ports, fmt.Sprintf("%v", p))
				}
			}
		}

		if len(ports) != 1 {
			return errors.Errorf("found %d ports, cannot infer application port, please specify on the command line",
				len(ports))
		} else {
			app_port = ports[0]
		}
	} else {
		app_port = fmt.Sprintf("%v", port)
	}

	license_key, _ := apictl.Flags().GetString("license-key")

	// inject the sidecar container
	podSpec["containers"] = append(typecastList(podSpec["containers"]), map[string]interface{}{
		"name":    "traffic-sidecar",
		"image":   getenvDefault("AES_IMAGE", "quay.io/datawire/aes:"+Version),
		"command": []string{"app-sidecar"},
		"env": []map[string]string{
			{"name": "APPNAME", "value": res.QName()},
			{"name": "APPPORT", "value": app_port},
			{"name": "AMBASSADOR_LICENSE_KEY", "value": license_key},
		},
		"ports": []map[string]interface{}{
			{"containerPort": 9900},
		},
	})

	return nil
}

func mungeService(res k8s.Resource) error {
	spec := res.Spec()
	iportSpecs, ok := spec["ports"]
	if !ok {
		return errors.Errorf("No ports found for service: %s", service)
	}
	portSpecs := iportSpecs.([]interface{})
	for _, iportSpec := range portSpecs {
		portSpec := iportSpec.(map[string]interface{})
		targetPort := portSpec["targetPort"]
		if targetPort == port {
			portSpec["targetPort"] = 9900
			return nil
		}
	}

	return errors.Errorf("service %s has no targetPort of %d", service, port)
}

var intercept = &cobra.Command{
	Use:   "intercept [flags] <name>",
	Short: "Intercept the traffic for a given deployment",
	Args:  cobra.ExactArgs(1),
	RunE:  doIntercept,
}

func init() {
	traffic.AddCommand(intercept)
	intercept.Flags().StringVarP(&name, "name", "n", "", "header name to match (:path, :method, and :authority are also available)")
	intercept.Flags().StringVarP(&match, "match", "m", "", "a regular expression to match")
	intercept.Flags().StringVarP(&target, "target", "t", "", "the [<host>:]<port> to forward to")
	intercept.MarkFlagRequired("target")
}

// GetFreePort asks the kernel for a free open port that is ready to use.
// Similar to telepresence.utilities.find_free_port()
func GetFreePort() (int, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var operr error
			fn := func(fd uintptr) {
				operr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			}
			if err := c.Control(fn); err != nil {
				return err
			}
			return operr
		},
	}
	l, err := lc.Listen(context.Background(), "tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

var apiPort int
var inboundPort int

var name string
var match string
var target string

func doIntercept(cmd *cobra.Command, args []string) error {
	if err := licenseClaims.RequireFeature(licensekeys.FeatureTraffic); err != nil {
		return err
	}

	var err error
	apiPort, err = GetFreePort()
	if err != nil {
		return errors.Wrap(err, "get free port for API")
	}
	inboundPort, err = GetFreePort()
	if err != nil {
		return errors.Wrap(err, "get free port for inbound")
	}
	info := k8s.NewKubeInfo("", "", "")
	kargs := fmt.Sprintf("port-forward service/telepresence-proxy %d:8022 %d:8081", inboundPort, apiPort)
	kubectl, err := info.GetKubectl(kargs)
	if err != nil {
		return err
	}
	pf := tpu.NewKeeper("KPF", "kubectl "+kubectl)
	kubectl, err = info.GetKubectl("describe service/telepresence-proxy deployment/telepresence-proxy")
	if err != nil {
		return err
	}
	pf.Inspect = "kubectl " + kubectl
	pf.Start()
	defer pf.Stop()

	time.Sleep(500 * time.Millisecond)

	remote_port, err := icept(args[0], name, match)
	if err != nil {
		return errors.Wrap(err, "launch interceptor")
	}
	log.Printf("ICP: remote port %s", remote_port)
	defer func() {
		log.Printf("ICP: %s", cleanup(args[0], remote_port))
	}()

	iremote_port, err := strconv.Atoi(remote_port)
	if err != nil {
		return errors.Wrapf(err, "parse number from interceptor: %s", remote_port)
	}
	go func() {
		for {
			time.Sleep(5 * time.Second)
			_, err := json_request(args[0], "POST", map[string]interface{}{
				"port": iremote_port,
			})
			if err != nil {
				// FIXME: Crash leaks subprocesses. Note that replacing Fatalf
				// with Panicf does not fix this, as a panic in this goroutine
				// will not fire deferred functions in other goroutines.
				log.Fatalf("ICP: unable to renew port %s: %v", remote_port, err)
			}
		}
	}()

	if !strings.Contains(target, ":") {
		target = fmt.Sprintf("127.0.0.1:%s", target)
	}

	ssh := tpu.NewKeeper("SSH", "ssh -C -N -oConnectTimeout=5 -oExitOnForwardFailure=yes "+
		fmt.Sprintf("-oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null telepresence@localhost -p %d ", inboundPort)+
		fmt.Sprintf("-R %s:%s", remote_port, target))
	ssh.Start()
	defer ssh.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Intercept is running. Press Ctrl-C/Ctrl-Break to quit.")

	log.Printf("ICP: %v", <-signalChan)
	return nil
}

func icept(name, header, regex string) (string, error) {
	for {
		result, err := json_request(name, "POST", map[string]interface{}{
			"name": name,
			"patterns": []map[string]interface{}{
				{"name": header, "regex_match": regex},
			},
		})
		if _, is_fatal := err.(*FatalError); is_fatal {
			return "", err
		} else if err != nil {
			log.Printf("ICP: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		return result, nil
	}
}

func cleanup(name, port string) string {
	result, err := request(name, "DELETE", []byte(port))
	if err != nil {
		return err.Error()
	} else {
		return result
	}
}

func json_request(name string, method string, data interface{}) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return request(name, method, encoded)
}

type FatalError struct {
	s string
}

func (e *FatalError) Error() string {
	return e.s
}

var client = &util.SimpleClient{Client: &http.Client{}}

func request(name string, method string, data []byte) (result string, err error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/intercept/%s", apiPort, name)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	body, err := client.DoBodyBytes(req, func(resp *http.Response, body []byte) (err error) {
		if resp.StatusCode == 404 {
			return &FatalError{fmt.Sprintf("no such deployment: %s", name)}
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
			return fmt.Errorf("%s: %s", http.StatusText(resp.StatusCode), string(body))
		}
		return nil
	})
	result = string(body)
	return
}
