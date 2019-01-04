package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/datawire/teleproxy/pkg/k8s"
	"github.com/datawire/teleproxy/pkg/tpu"
	"github.com/spf13/cobra"
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
	Run:   doInitialize,
}

func init() {
	traffic.AddCommand(initialize)
}

const (
	TRAFFIC_MANAGER = `
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
        image: quay.io/datawire/ambassador-ratelimit:proxy-0.0.5
        ports:
        - name: sshd
          containerPort: 8022
`
)

func doInitialize(cmd *cobra.Command, args []string) {
	info, err := k8s.NewKubeInfo("", "", "")
	die(err)

	apply := tpu.NewKeeper("KAP", "kubectl "+info.GetKubectl("apply -f -"))
	apply.Input = TRAFFIC_MANAGER
	apply.Limit = 1
	apply.Start()
	apply.Wait()

	w := k8s.NewWaiter(k8s.NewClient(info).Watcher())
	err = w.Add("service/telepresence-proxy")
	die(err)
	err = w.Add("deployment/telepresence-proxy")
	die(err)
	if !w.Wait(30) {
		os.Exit(1)
	}
}

var inject = &cobra.Command{
	Use:   "inject",
	Short: "Inject the traffic sidecar into a deployment",
	Run:   doInject,
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

func doInject(cmd *cobra.Command, args []string) {
	for _, arg := range args {
		resources, err := k8s.LoadResources(arg)
		die(err)
		for _, res := range resources {
			if strings.ToLower(res.Kind()) == "deployment" && res.Name() == deployment {
				munge(res)
			}
			if strings.ToLower(res.Kind()) == "service" && res.Name() == service {
				mungeService(res)
			}
		}
		bytes, err := k8s.MarshalResources(resources)
		die(err)
		fmt.Print(string(bytes))
	}
}

func munge(res k8s.Resource) {
	spec := res.Spec()
	podSpec := spec["template"].(map[string]interface{})["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]interface{})

	var app_port string
	if port == 0 {
		var ports []string
		for _, c := range containers {
			iportSpecs, ok := c.(map[string]interface{})["ports"]
			if !ok {
				continue
			}

			portSpecs := iportSpecs.([]interface{})

			for _, portSpec := range portSpecs {
				p, ok := portSpec.(map[string]interface{})["containerPort"]
				if ok {
					ports = append(ports, fmt.Sprintf("%v", p))
				}
			}
		}

		if len(ports) != 1 {
			die(fmt.Errorf("found %d ports, cannot infer application port, please specify on the command line",
				len(ports)))
		} else {
			app_port = ports[0]
		}
	} else {
		app_port = fmt.Sprintf("%v", port)
	}

	blah := make(map[string]interface{})
	blah["name"] = "traffic-sidecar"
	blah["image"] = "quay.io/datawire/ambassador-ratelimit:sidecar-0.0.5"
	blah["env"] = []map[string]string{
		{"name": "APPNAME", "value": res.QName()},
		{"name": "APPPORT", "value": app_port},
	}
	blah["ports"] = []map[string]interface{}{
		{"containerPort": 9900},
	}

	containers = append(containers, blah)
	podSpec["containers"] = containers
}

func mungeService(res k8s.Resource) {
	spec := res.Spec()
	iportSpecs, ok := spec["ports"]
	if !ok {
		die(fmt.Errorf("No ports found for service: %s", service))
	}
	portSpecs := iportSpecs.([]interface{})
	for _, iportSpec := range portSpecs {
		portSpec := iportSpec.(map[string]interface{})
		targetPort := portSpec["targetPort"]
		if targetPort == port {
			portSpec["targetPort"] = 9900
			return
		}
	}

	die(fmt.Errorf("service %s has no targetPort of %d", service, port))
}

var intercept = &cobra.Command{
	Use:   "intercept",
	Short: "Intercept the traffic for a given deployment",
	Args:  cobra.MinimumNArgs(1),
	Run:   doIntercept,
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
	l, err := lc.Listen(context.Background(), "tcp", "localhost:0")
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

func doIntercept(cmd *cobra.Command, args []string) {
	var err error
	apiPort, err = GetFreePort()
	die(err)
	inboundPort, err = GetFreePort()
	die(err)
	info, err := k8s.NewKubeInfo("", "", "")
	die(err)
	kargs := fmt.Sprintf("port-forward service/telepresence-proxy %d:8022 %d:8081", inboundPort, apiPort)
	pf := tpu.NewKeeper("KPF", "kubectl "+info.GetKubectl(kargs))
	pf.Inspect = "kubectl " + info.GetKubectl("describe service/telepresence-proxy deployment/telepresence-proxy")
	pf.Start()
	defer pf.Stop()

	time.Sleep(500 * time.Millisecond)

	remote_port := icept(args[0], name, match)
	log.Printf("ICP: remote port %s", remote_port)
	defer func() {
		log.Printf("ICP: %s", cleanup(args[0], remote_port))
	}()

	iremote_port, err := strconv.Atoi(remote_port)
	die(err)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			_, err := json_request(args[0], "POST", map[string]interface{}{
				"port": iremote_port,
			})
			if err != nil {
				log.Fatalf("ICP: unable to renew port %s: %v", remote_port, err)
			} else {
				log.Printf("ICP: renewed port %s", remote_port)
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

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("ICP: %v", <-signalChan)
}

func icept(name, header, regex string) string {
	for {
		result, err := json_request(name, "POST", map[string]interface{}{
			"name": name,
			"patterns": []map[string]interface{}{
				{"name": header, "regex_match": regex},
			},
		})
		if err != nil {
			f, ok := err.(*FatalError)
			if ok {
				die(f)
			} else {
				log.Printf("ICP: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
		}
		return result
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

func request(name string, method string, data []byte) (result string, err error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/intercept/%s", apiPort, name)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return "", &FatalError{fmt.Sprintf("no such deployment: %s", name)}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		err = fmt.Errorf("%s: %s", http.StatusText(resp.StatusCode), string(body))
		return
	}
	result = string(body)
	return
}
