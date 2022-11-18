package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/kubeapply"
)

var tmpl = template.Must(template.
	New("docker-registry.yaml").
	Parse(`---
apiVersion: v1
kind: Namespace
metadata:
  name: docker-registry
---
apiVersion: v1
kind: Service
metadata:
  namespace: docker-registry
  name: registry
spec:
  type: NodePort
  selector:
    app: registry
  ports:
    - port: 5000
      nodePort: 31000
---
apiVersion: apps/v1
# XXX: Avoid using a StatefulSet if possible, because kubeapply
# doesn't know how to wait for them.
kind: {{ if eq .Storage "pvc" }}StatefulSet{{ else }}Deployment{{ end }}
metadata:
  namespace: docker-registry
  name: registry
spec:
  replicas: 1
{{ if eq .Storage "pvc" }} # XXX: StatefulSet
  serviceName: registry
{{ end }}
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      name: registry
      labels:
        app: registry
    spec:
      containers:
        - name: registry
          image: docker.io/library/registry:2
          ports:
            - containerPort: 5000
          volumeMounts:
            - mountPath: /var/lib/registry
              name: registry-data
      volumes:
        - name: registry-data
{{ if eq .Storage "pvc" | not }}
          # On Kubeception clusters, there is only 1 node, so a
          # hostPath is fine.
          hostPath:
            path: /var/lib/registry
{{ else }}
          persistentVolumeClaim:
            claimName: registry-data
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: registry-data
  namespace: docker-registry
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
{{ end }}
`))

func main() {
	argparser := &cobra.Command{
		Use:           os.Args[0],
		Short:         "Manage a private in-cluster registry",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	argparser.AddCommand(func() *cobra.Command {
		var (
			argStorage string
		)
		subparser := &cobra.Command{
			Use:   "up",
			Short: "Initialize the registry, and create a port-forward to it",
			Args:  cobra.ExactArgs(0),
			RunE: func(cobraCmd *cobra.Command, _ []string) error {
				var kpfTarget string
				switch argStorage {
				case "pvc":
					kpfTarget = "statefulset/registry"
				case "hostPath":
					kpfTarget = "deployment/registry"
				default:
					return errors.Errorf("invalid --storage=%q: must be one of 'pvc' or 'hostPath'", argStorage)
				}

				kubeclient, err := kates.NewClient(kates.ClientConfig{})
				if err != nil {
					return err
				}

				// Part 1: Apply the YAML
				//
				// pkg/kubeapply is annoyingly oriented around actual physical files >:(
				tmpdir, err := ioutil.TempDir("", filepath.Base(os.Args[0]))
				if err != nil {
					return err
				}
				defer os.RemoveAll(tmpdir)
				yamlFile, err := os.OpenFile(filepath.Join(tmpdir, "docker-registry.yaml"), os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					return err
				}
				err = tmpl.Execute(yamlFile, map[string]interface{}{
					"Storage": argStorage,
				})
				yamlFile.Close()
				if err != nil {
					return err
				}
				err = kubeapply.Kubeapply(
					cobraCmd.Context(), // context
					kubeclient,         // kubeclient
					time.Minute,        // perPhaseTimeout
					false,              // debug
					false,              // dryRun
					yamlFile.Name(),    // files
				)
				if err != nil {
					return err
				}

				// Part 2: Set up the port-forward
				cmd := exec.Command(
					"kubectl",
					"port-forward",
					"--namespace=docker-registry",
					kpfTarget,
					"31000:5000")
				cmd.Stdout, err = os.OpenFile(filepath.Join(os.TempDir(), filepath.Base(os.Args[0])+".log"), os.O_CREATE|os.O_WRONLY, 0666)
				if err != nil {
					return err
				}
				cmd.Stderr = cmd.Stdout
				if err := cmd.Start(); err != nil {
					return err
				}
				for {
					_, httpErr := http.Get("http://localhost:31000/")
					if httpErr == nil {
						fmt.Fprintln(os.Stderr, "port-forward ready")
						break
					} else {
						fmt.Fprintln(os.Stderr, "waiting for port-forward to become ready...")
						time.Sleep(time.Second)
					}
				}
				return nil
			},
		}
		subparser.Flags().StringVar(&argStorage, "storage", "", "Which type of storage to use ('pvc' or 'hostPath')")
		return subparser
	}())
	argparser.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Shut down the port-forward to the registry",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("killall", "kubectl") // XXX
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	})
	if err := argparser.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
