package reproducer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dexec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract [<out-file>]",
	Short: "extract a redacted set of Ambassador Edge Stack inputs/configuration and logs/debug",
	Long: `The extract subcommand is inteded to help extract as much info as possible from a source cluster to aid in creation of a reproducer. This source info is redacted and then bundled up in a single archive for ease of uploading.

The extract subcommand is designed to be run from both outside the cluster and from in the ambassador pod itself. In each case it will capture as much as it can, however it is preferrable to run it from outside the cluster as it will likely have more expansive rbac privileges and therefore be able to capture more relevant details.

Currently the extract command when run with sufficient rbac privileges captures:

  - The previous and current logs for all ambassador pods.
  - The output of grab-snapshots for all ambassador pods.
  - Additional resources not included in the snapshot.
    + All apro resources.
    + All pod info/states.
    + The cluster Event log.
  - The environment variables for the ambassador pods (with AUTH and PASSWORDs redacted).
`,
	Args: cobra.RangeArgs(0, 1),
	RunE: extract,
}

func extract(cmd *cobra.Command, args []string) error {
	var filename string
	if len(args) > 0 {
		filename = args[0]
	} else {
		filename = fmt.Sprintf("extraction-%s.tgz", time.Now().Format(time.RFC3339))
	}

	ctx := cmd.Context()
	cli, err := kates.NewClient(kates.ClientConfig{})
	if err != nil {
		return errors.Wrapf(err, "initializing kubernetes client")
	}

	ex := NewExtraction(cli)

	// Find interesting pods.
	pods := ex.ListAmbassadorPods(ctx)

	// Kick off async log capture for those pods.
	podLogsFunc := ex.CaptureLogs(ctx, pods)

	// Capture snapshots from pods
	err = ex.CaptureRemoteSnapshots(ctx, pods)
	if err != nil {
		return err
	}

	// Capture interesting resources.
	err = ex.CaptureResources(ctx)
	if err != nil {
		return err
	}

	// Capture the environment if we are inside the cluster.
	if kates.InCluster() {
		ex.CaptureEnviron(ctx)
		ex.CaptureSnapshot(ctx)
	}

	// Save all the results in a tarball.
	return ex.WriteArchive(ctx, filename, podLogsFunc())
}

type PodLogs = map[string][]kates.LogEvent

type Extraction struct {
	client        *kates.Client
	ExtractionLog []*LogEntry           // A log of the extraction process itsef.
	Snapshots     map[string][]byte     // Snapshots from all pods in the cluster.
	Resources     []*kates.Unstructured // Interesting resources in the cluster that may not be included in snapshots.
	Environ       map[string]string     // Capture the environment if we are invoked in the cluster.
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Error     error     `json:"error,omitempty"`
}

func NewExtraction(client *kates.Client) *Extraction {
	return &Extraction{client: client, Snapshots: map[string][]byte{}}
}

func (ex *Extraction) add(entry *LogEntry) {
	ex.ExtractionLog = append(ex.ExtractionLog, entry)
}

func (ex *Extraction) Printf(ctx context.Context, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s\n", fmt.Sprintf(format, args...))
	ex.add(&LogEntry{Timestamp: time.Now(), Message: fmt.Sprintf(format, args...)})
}

func (ex *Extraction) Warnf(ctx context.Context, err error, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s: %+v\n", fmt.Sprintf(format, args...), err)
	ex.add(&LogEntry{Timestamp: time.Now(), Message: fmt.Sprintf(format, args...), Error: err})
}

// ListAmbassadorPods will search the entire cluster for ambassador pods resulting from either a
// helm based or manifest based install. It does this by using the service=ambassador labeling used
// by the manifests and the product=aes labeling used by the helm charts. This may get more pods
// than just edge-stack, e.g. it may pull in the redis logs since they have the product=aes label,
// but that is ok we would rather grab more debugging info than less.
func (ex *Extraction) ListAmbassadorPods(ctx context.Context) []*kates.Pod {
	var result []*kates.Pod
	for _, sel := range []string{"service=ambassador", "product=aes"} {
		var pods []*kates.Pod
		err := ex.client.List(ctx, kates.Query{Kind: "Pod", Namespace: kates.NamespaceAll, LabelSelector: sel}, &pods)
		if err != nil {
			ex.Warnf(ctx, err, "error listing pods, no logs will be available")
			continue
		}
		result = append(result, pods...)
	}

	var podNames []string
	for _, p := range result {
		podNames = append(podNames, QName(p))
	}
	sort.Strings(podNames)
	if len(result) > 0 {
		ex.Printf(ctx, "found ambassador pods: %s", strings.Join(podNames, ", "))
	} else {
		ex.Printf(ctx, "unable to find ambassador pods")
	}
	return result
}

// CaptureLogs will capture the current and previous logs from all the listed pods. It operates
// asynchronously and returns a function that can be used to access the final result (i.e. a poor
// mans future).
func (ex *Extraction) CaptureLogs(ctx context.Context, pods []*kates.Pod) func() PodLogs {
	previousEvents := make(chan kates.LogEvent)
	currentEvents := make(chan kates.LogEvent)
	wg := sync.WaitGroup{}
	byID := map[string]string{}
	for _, pod := range pods {
		byID[string(pod.GetUID())] = QName(pod)
		err := ex.client.PodLogs(ctx, pod, &kates.PodLogOptions{Previous: true}, previousEvents)
		if err != nil {
			ex.Warnf(ctx, err, "error listing previous logs for pod %s in namespaces %s", pod.Name, pod.Namespace)
		} else {
			wg.Add(1)
		}
		err = ex.client.PodLogs(ctx, pod, &kates.PodLogOptions{}, currentEvents)
		if err != nil {
			ex.Warnf(ctx, err, "error listing current logs for pod %s in namespaces %s", pod.Name, pod.Namespace)
		} else {
			wg.Add(1)
		}
	}
	podLogs := PodLogs{}
	go func() {
		for {
			var ev kates.LogEvent
			var name string
			select {
			case ev = <-previousEvents:
				name = fmt.Sprintf("%s:previous", byID[ev.PodID])
			case ev = <-currentEvents:
				name = byID[ev.PodID]
			}
			podLogs[name] = append(podLogs[name], ev)
			if ev.Closed {
				wg.Done()
			}
		}
	}()

	return func() PodLogs {
		wg.Wait()
		return podLogs
	}
}

// CaptureRemoteSnapshots will exec grab-snapshots on all the supplied pods and capture the resulting tarballs.
func (ex *Extraction) CaptureRemoteSnapshots(ctx context.Context, pods []*kates.Pod) error {
	for _, p := range pods {
		cmd := dexec.CommandContext(ctx, "kubectl", "exec", "-i", "-n", p.GetNamespace(), p.GetName(), "--", "grab-snapshots", "-o", "/tmp/sanitized.tgz")
		cmd.DisableLogging = true
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			ex.Warnf(ctx, err, "error grabbing snapshot for pod %s", QName(p))
			continue
		}

		cmd = dexec.CommandContext(ctx, "kubectl", "cp", "-n", p.GetNamespace(), fmt.Sprintf("%s:/tmp/sanitized.tgz", p.GetName()), "/dev/stdout")
		cmd.DisableLogging = true
		cmd.Stderr = nil
		snapshot, err := cmd.Output()
		if err != nil {
			ex.Warnf(ctx, err, "error copying snapshot for pod %s", QName(p))
			continue
		}
		ex.Snapshots[QName(p)] = snapshot
	}

	return nil
}

// CaptureResources will capture and sanitize as many resources as permitted by the RBAC of the
// system account running the extraction. The code is careful to redact both secrets and config maps
// as well as the enviornment variables of any unrecognized deployments. For ambassador deployments
// only the environment variables that contain AUTH and/or PASSWORD are redacted.
func (ex *Extraction) CaptureResources(ctx context.Context) error {
	preferredResources, err := ex.client.ServerPreferredResources()
	if err != nil {
		return errors.Wrapf(err, "querying server resources")
	}

	for _, r := range preferredResources {
		hasList := false
		for _, v := range r.Verbs {
			if v == "list" {
				hasList = true
			}
		}
		if hasList {
			ex.capture(ctx, kates.Query{Kind: r.Kind, Namespace: kates.NamespaceAll})
		}
	}

	ex.Printf(ctx, "extracted %d total resources", len(ex.Resources))
	return nil
}

func (ex *Extraction) capture(ctx context.Context, query kates.Query) {
	var rsrcs []*kates.Unstructured
	err := ex.client.List(ctx, query, &rsrcs)
	if err != nil {
		ex.Warnf(ctx, err, "error extracting resource %s", query.Kind)
		return
	}

	sanitized := []*kates.Unstructured{}
	for _, r := range rsrcs {
		s := ex.callSanitize(ctx, r)
		if s != nil {
			sanitized = append(sanitized, s)
		}
	}

	ex.Printf(ctx, "extracted %d of %d %s", len(sanitized), len(rsrcs), query.Kind)
	ex.Resources = append(ex.Resources, sanitized...)
}

func (ex *Extraction) callSanitize(ctx context.Context, resource *kates.Unstructured) *kates.Unstructured {
	obj, err := kates.NewObjectFromUnstructured(resource)
	if err != nil {
		ex.Printf(ctx, "error sanitizing object: %+v", err)
		return nil
	}

	obj = ex.sanitize(ctx, obj)

	result, err := kates.NewUnstructuredFromObject(obj)
	if err != nil {
		log.Printf("error converting resource to Unstructured: %+v", err)
		return nil
	}
	return result
}

func (ex *Extraction) sanitize(ctx context.Context, object kates.Object) kates.Object {
	// Don't capture secrets and don't capture ConfigMaps because the latter often has secrets.
	switch obj := object.(type) {
	case *kates.Secret:
		if obj.Type == kates.SecretTypeServiceAccountToken {
			return nil
		}

		ex.Printf(ctx, "redacting secret %s", QName(obj))
		data := map[string][]byte{}
		for k := range obj.Data {
			data[k] = []byte("<redacted>")
		}
		obj.Data = data
		obj.StringData = nil
	case *kates.ConfigMap:
		ex.Printf(ctx, "redacting configmap %s", QName(obj))
		data := map[string]string{}
		for k := range obj.Data {
			data[k] = "<redacted>"
		}
		obj.Data = data
		obj.BinaryData = nil
	case *kates.Deployment:
		for _, c := range obj.Spec.Template.Spec.Containers {
			filtered := []kates.EnvVar{}
			for _, e := range c.Env {
				copy := e
				if e.Value != "" {
					if isAmbassadorResource(obj) {
						if strings.Contains(e.Name, "AUTH") || strings.Contains(e.Name, "PASSWORD") {
							ex.Printf(ctx, "redacting env var %s", e.Name)
							copy.Value = "<redacted>"
						}
					} else {
						ex.Printf(ctx, "redacting env var %s", e.Name)
						copy.Value = "<redacted>"
					}
				}
				filtered = append(filtered, copy)
			}
			c.Env = filtered
		}
	}

	return object
}

func isAmbassadorResource(object kates.Object) bool {
	labels := object.GetLabels()
	if labels["product"] == "aes" {
		return true
	}

	return false
}

// CaptureEnviron captures the environment while redacting any secrets.
func (ex *Extraction) CaptureEnviron(ctx context.Context) {
	ex.Environ = map[string]string{}
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			ex.Printf(ctx, "unable to split os.Environ() result %v", e)
			continue
		}
		k := parts[0]
		v := parts[1]
		if strings.Contains(k, "AUTH") || strings.Contains(k, "PASSWORD") {
			v = "<redacted>"
			ex.Printf(ctx, "redacting %s environmen variable", k)
		}
		ex.Environ[k] = v
	}
	ex.Printf(ctx, "extracted %d environment variables", len(ex.Environ))
}

// CaptureSnapshot captures the local snapshot if we are in cluster.
func (ex *Extraction) CaptureSnapshot(ctx context.Context) {
	cmd := dexec.CommandContext(ctx, "grab-snapshots", "-o", "/dev/stdout")
	cmd.DisableLogging = true
	cmd.Stderr = nil
	snapshot, err := cmd.Output()
	if err != nil {
		ex.Warnf(ctx, err, "error extracting local snapshot")
		return
	}
	ex.Snapshots["local"] = snapshot
}

// WriteArchive saves all the extracted info into a tarball.
func (ex *Extraction) WriteArchive(ctx context.Context, filename string, podLogs PodLogs) error {
	manifests, err := marshalManifests(ex.Resources)
	if err != nil {
		return errors.Wrapf(err, "marshalling resources")
	}

	logTotal := 0
	for k, v := range podLogs {
		logTotal += len(v)
		ex.Printf(ctx, "extracted %d log entries from pod %s", len(v), k)
	}
	ex.Printf(ctx, "extracted %d total log entries", logTotal)

	out, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "creating output")
	}
	ex.Printf(ctx, "created %s", filename)
	defer func() {
		out.Close()
		ex.Printf(ctx, "closed %s", filename)
	}()

	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	archive := func(name string, content []byte) error {
		ex.Printf(ctx, "%s: adding %s (%d bytes)", filename, name, len(content))
		header := &tar.Header{
			Name:    name,
			Size:    int64(len(content)),
			Mode:    0777,
			ModTime: time.Now(),
		}

		err = tw.WriteHeader(header)
		if err != nil {
			return errors.Wrapf(err, "writing archive header %s", name)
		}
		_, err = io.Copy(tw, bytes.NewReader(content))
		if err != nil {
			return errors.Wrapf(err, "writing archive entry %s", name)
		}

		return nil
	}

	err = archive("manifests.yaml", manifests)
	if err != nil {
		return err
	}

	archiveJson := func(name string, value interface{}) error {
		bytes, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return archive(name, bytes)
	}

	err = archiveJson("pods.log", podLogs)
	if err != nil {
		return err
	}

	for k, v := range ex.Snapshots {
		err = archive(fmt.Sprintf("%s.snapshot.tgz", k), v)
		if err != nil {
			return err
		}
	}

	if kates.InCluster() {
		err = archiveJson("environ.json", ex.Environ)
		if err != nil {
			return err
		}
	}

	return archiveJson("extraction.log", ex.ExtractionLog)
}

func QName(obj kates.Object) string {
	return fmt.Sprintf("%s.%s", obj.GetName(), obj.GetNamespace())
}
