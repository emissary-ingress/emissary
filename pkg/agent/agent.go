package agent

import (
	"context"
	"encoding/json"
	"fmt"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v2"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

const defaultMinReportPeriod = 30 * time.Second
const cloudConnectTokenKey = "CLOUD_CONNECT_TOKEN"

type Comm interface {
	Close() error
	Report(context.Context, *agent.Snapshot, string) error
	Directives() <-chan *agent.Directive
	StreamMetrics(context.Context, *agent.StreamMetricsMessage, string) error
}

type atomicBool struct {
	mutex sync.Mutex
	value bool
}

func (ab *atomicBool) Value() bool {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	return ab.value
}

func (ab *atomicBool) Set(v bool) {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	ab.value = v
}

// Agent is the component that talks to the DCP Director, which is a cloud
// service run by Datawire.
type Agent struct {
	// Connectivity to the Director

	comm                  Comm
	connInfo              *ConnInfo
	agentID               *agent.Identity
	newDirective          <-chan *agent.Directive
	ambassadorAPIKeyMutex sync.Mutex
	ambassadorAPIKey      string
	directiveHandler      DirectiveHandler
	// store what the initial value was in the env var so we can set the ambassadorAPIKey value
	// (^^Above) if the configmap and/or secret get deleted.
	ambassadorAPIKeyEnvVarValue string
	connAddress                 string

	// State managed by the director via the retriever

	reportingStopped bool          // Did the director say don't report?
	minReportPeriod  time.Duration // How often can we Report?
	lastDirectiveID  string

	// The state of reporting

	reportToSend   *agent.Snapshot // Report that's ready to send
	reportRunning  atomicBool      // Is a report being sent right now?
	reportComplete chan error      // Report() finished with this error

	// current cluster state of core resources
	coreStore *coreStore

	// apiDocsStore holds OpenAPI documents from cluster Mappings
	apiDocsStore *APIDocsStore

	// rolloutStore holds Argo Rollouts state from cluster
	rolloutStore *RolloutStore
	// applicationStore holds Argo Applications state from cluster
	applicationStore *ApplicationStore

	// config map/secret information
	// agent namespace is... the namespace the agent is running in.
	// but more importantly, it's the namespace that the config resource lives in (which is
	// either a ConfigMap or Secret)
	agentNamespace string
	// Name of the k8s ConfigMap or Secret the CLOUD_CONNECT_TOKEN exists on. We're supporting
	// both Secrets and ConfigMaps here because it is likely in an enterprise cluster, the RBAC
	// for secrets is locked down to Ops folks only, and we want to make it easy for regular ol'
	// engineers to give this whole service catalog thing a go
	agentCloudResourceConfigName string

	// Field selector for the k8s resources that the agent watches
	agentWatchFieldSelector string
}

func getEnvWithDefault(envVarKey string, defaultValue string) string {
	value := os.Getenv(envVarKey)
	if value == "" {
		value = defaultValue
	}
	return value
}

// New returns a new Agent.
func NewAgent(directiveHandler DirectiveHandler) *Agent {
	reportPeriodFromEnv := os.Getenv("AGENT_REPORTING_PERIOD")
	var reportPeriod time.Duration
	if reportPeriodFromEnv != "" {
		reportPeriod, err := time.ParseDuration(reportPeriodFromEnv)
		if err != nil {
			reportPeriod = defaultMinReportPeriod
		} else {
			reportPeriod = MaxDuration(defaultMinReportPeriod, reportPeriod)
		}
	} else {
		reportPeriod = defaultMinReportPeriod
	}
	if directiveHandler == nil {
		directiveHandler = &BasicDirectiveHandler{DefaultMinReportPeriod: defaultMinReportPeriod}
	}

	return &Agent{
		minReportPeriod:  reportPeriod,
		reportComplete:   make(chan error),
		ambassadorAPIKey: os.Getenv(cloudConnectTokenKey),
		// store this same value in a different variable, so that if ambassadorAPIKey gets
		// changed by some other configuration, we know what to change it back to. See
		// comment on the struct for more detail
		ambassadorAPIKeyEnvVarValue:  os.Getenv(cloudConnectTokenKey),
		connAddress:                  os.Getenv("RPC_CONNECTION_ADDRESS"),
		agentNamespace:               getEnvWithDefault("AGENT_NAMESPACE", "ambassador"),
		agentCloudResourceConfigName: getEnvWithDefault("AGENT_CONFIG_RESOURCE_NAME", "ambassador-agent-cloud-token"),
		directiveHandler:             directiveHandler,
		reportRunning:                atomicBool{value: false},
		agentWatchFieldSelector:      getEnvWithDefault("AGENT_WATCH_FIELD_SELECTOR", "metadata.namespace!=kube-system"),
	}
}

func (a *Agent) StopReporting(ctx context.Context) {
	dlog.Debugf(ctx, "stop reporting: %t -> true", a.reportingStopped)
	a.reportingStopped = true
}

func (a *Agent) StartReporting(ctx context.Context) {
	dlog.Debugf(ctx, "stop reporting: %t -> false", a.reportingStopped)
	a.reportingStopped = false
}

func (a *Agent) SetMinReportPeriod(ctx context.Context, dur time.Duration) {
	dlog.Debugf(ctx, "minimum report period %s -> %s", a.minReportPeriod, dur)
	a.minReportPeriod = dur
}

func (a *Agent) SetLastDirectiveID(ctx context.Context, id string) {
	dlog.Debugf(ctx, "setting last directive ID %s", id)
	a.lastDirectiveID = id
}

func getAmbSnapshotInfo(url string) (*snapshotTypes.Snapshot, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rawSnapshot, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &snapshotTypes.Snapshot{}
	err = json.Unmarshal(rawSnapshot, ret)

	return ret, err
}

func parseAmbassadorAdminHost(rawurl string) (string, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	return url.Hostname(), nil

}

func getAPIKeyValue(configValue string, configHadValue bool) string {
	if configHadValue {
		return configValue
	}
	return ""
}

// Handle change to the ambassadorAPIKey that we auth to the agent with
// in order of importance: secret > configmap > environment variable
// so if a secret exists, read from that. then, check if a config map exists, and read the value
// from that. If neither a secret or a configmap exists, use the value from the environment that we
// stored on startup.
func (a *Agent) handleAPIKeyConfigChange(ctx context.Context, secrets []kates.Secret, configMaps []kates.ConfigMap) {
	// reset the connection so we use a new api key (or break the connection if the api key was
	// unset). The agent will reset the connection the next time it tries to send a report
	resetComm := func(newKey string, oldKey string, a *Agent) {
		if newKey != oldKey {
			a.ClearComm()
		}
	}
	prevKey := a.ambassadorAPIKey
	// first, check if we have a secret, since we want that value to take if we
	// can get it.
	// there _should_ only be one secret here, but we're going to loop and check that the object
	// meta matches what we expect
	for _, secret := range secrets {
		if secret.GetName() == a.agentCloudResourceConfigName && secret.GetNamespace() == a.agentNamespace {
			connTokenBytes, ok := secret.Data[cloudConnectTokenKey]
			connToken := string(connTokenBytes)
			dlog.Infof(ctx, "Setting cloud connect token from secret")
			a.ambassadorAPIKey = getAPIKeyValue(connToken, ok)
			resetComm(a.ambassadorAPIKey, prevKey, a)
			return
		}
	}
	// then, if we don't have a secret, we check for a config map
	// there _should_ only be one config here, but we're going to loop and check that the object
	// meta matches what we expect
	for _, cm := range configMaps {
		if cm.GetName() == a.agentCloudResourceConfigName && cm.GetNamespace() == a.agentNamespace {
			connTokenBytes, ok := cm.Data[cloudConnectTokenKey]
			connToken := string(connTokenBytes)
			dlog.Infof(ctx, "Setting cloud connect token from configmap")
			a.ambassadorAPIKey = getAPIKeyValue(connToken, ok)
			resetComm(a.ambassadorAPIKey, prevKey, a)
			return
		}
	}
	// so if we got here, we know something changed, but a config map
	// nor a secret exist, which means they never existed or they got
	// deleted. in this case, we fall back to the env var (which is
	// likely empty, so in that case, that is basically equivelant to
	// turning the agent "off")
	dlog.Infof(ctx, "Setting cloud connect token from environment")
	a.ambassadorAPIKeyMutex.Lock()
	defer a.ambassadorAPIKeyMutex.Unlock()
	a.ambassadorAPIKey = a.ambassadorAPIKeyEnvVarValue
	resetComm(a.ambassadorAPIKey, prevKey, a)
}

// Watch is the work performed by the main goroutine for the Agent. It processes
// Watt/Diag snapshots, reports to the Director, and executes directives from
// the Director.
func (a *Agent) Watch(ctx context.Context, snapshotURL string) error {
	client, err := kates.NewClient(kates.ClientConfig{})
	if err != nil {
		return err
	}
	dlog.Info(ctx, "Agent is running...")
	agentCMQuery := kates.Query{
		Namespace:     a.agentNamespace,
		Name:          "ConfigMaps",
		Kind:          "configmaps.",
		FieldSelector: fmt.Sprintf("metadata.name=%s", a.agentCloudResourceConfigName),
	}
	agentSecretQuery := kates.Query{
		Namespace:     a.agentNamespace,
		Name:          "Secrets",
		Kind:          "secrets.",
		FieldSelector: fmt.Sprintf("metadata.name=%s", a.agentCloudResourceConfigName),
	}
	configAcc, err := client.Watch(ctx, agentCMQuery, agentSecretQuery)
	if err != nil {
		return err
	}
	if err := a.waitForAPIKey(ctx, configAcc); err != nil {
		dlog.Errorf(ctx, "Error waiting for api key: %+v", err)
		return err
	}

	podQuery := kates.Query{
		Name:          "Pods",
		Kind:          "pods.",
		FieldSelector: a.agentWatchFieldSelector,
	}
	cmQuery := kates.Query{
		Name:          "ConfigMaps",
		Kind:          "configmaps.",
		FieldSelector: a.agentWatchFieldSelector,
	}
	deployQuery := kates.Query{
		Name:          "Deployments",
		Kind:          "deployments.",
		FieldSelector: a.agentWatchFieldSelector,
	}
	endpointQuery := kates.Query{
		Name:          "Endpoints",
		Kind:          "endpoints.",
		FieldSelector: a.agentWatchFieldSelector,
	}

	// If the user didn't setup RBAC to allow the agent to get pods, the watch will just return
	// no pods, log that it didn't have permission to get pods, and carry along.
	coreAcc, err := client.Watch(ctx, podQuery, cmQuery, deployQuery, endpointQuery)
	if err != nil {
		return err
	}

	ns := kates.NamespaceAll
	dc := NewDynamicClient(client.DynamicInterface(), NewK8sInformer)
	rolloutGvr, _ := schema.ParseResourceArg("rollouts.v1alpha1.argoproj.io")
	rolloutCallback := dc.WatchGeneric(ctx, ns, rolloutGvr)

	applicationGvr, _ := schema.ParseResourceArg("applications.v1alpha1.argoproj.io")
	applicationCallback := dc.WatchGeneric(ctx, ns, applicationGvr)

	return a.watch(ctx, snapshotURL, configAcc, coreAcc, rolloutCallback, applicationCallback)
}

type accumulator interface {
	Changed() chan struct{}
	FilteredUpdate(ctx context.Context, target interface{}, deltas *[]*kates.Delta, predicate func(*kates.Unstructured) bool) (bool, error)
}

func (a *Agent) waitForAPIKey(ctx context.Context, configAccumulator accumulator) error {
	isValid := func(un *kates.Unstructured) bool {
		return true
	}
	configSnapshot := struct {
		Secrets    []kates.Secret
		ConfigMaps []kates.ConfigMap
	}{}
	// wait until the user installs an api key
	for a.ambassadorAPIKey == "" {
		select {
		case <-ctx.Done():
			return nil
		case <-configAccumulator.Changed():
			updated, err := configAccumulator.FilteredUpdate(ctx, &configSnapshot, &[]*kates.Delta{}, isValid)
			if err != nil {
				return err
			}
			if !updated {
				continue
			}
			a.handleAPIKeyConfigChange(ctx, configSnapshot.Secrets, configSnapshot.ConfigMaps)
		case <-time.After(1 * time.Minute):
			dlog.Debugf(ctx, "Still waiting for api key")
		}
	}
	return nil
}

func (a *Agent) watch(ctx context.Context, snapshotURL string, configAccumulator accumulator, coreAccumulator accumulator, rolloutCallback <-chan *GenericCallback, applicationCallback <-chan *GenericCallback) error {
	var err error
	// for the watch
	// we're not watching CRDs or anything special, so i'm pretty sure it's okay just to say all
	// the pods are valid
	isValid := func(un *kates.Unstructured) bool {
		return true
	}
	ambHost, err := parseAmbassadorAdminHost(snapshotURL)
	if err != nil {
		// if we can't parse the host out of the url we won't be able to talk to ambassador
		// anyway
		return err
	}

	a.apiDocsStore = NewAPIDocsStore()
	applicationStore := NewApplicationStore()
	rolloutStore := NewRolloutStore()
	coreSnapshot := CoreSnapshot{}
	configSnapshot := struct {
		Secrets    []kates.Secret
		ConfigMaps []kates.ConfigMap
	}{}
	dlog.Info(ctx, "Beginning to watch and report resources to ambassador cloud")
	for {
		// Wait for an event
		select {
		case <-ctx.Done():
			return nil
			// just hardcode it so we wake every 1 second and check if we're ready to report
			// intentionally not waiting for agent.minReportPeriod seconds because then we may
			// never report if a bunch of directives keep coming in or pods change a
			// bunch
		case <-time.After(1 * time.Second):
			// just a ticker, this will fallthru to the snapshot getting thing
		case <-configAccumulator.Changed():
			updated, err := configAccumulator.FilteredUpdate(ctx, &configSnapshot, &[]*kates.Delta{}, isValid)
			if err != nil {
				return err
			}
			if !updated {
				continue
			}
			a.handleAPIKeyConfigChange(ctx, configSnapshot.Secrets, configSnapshot.ConfigMaps)
		case <-coreAccumulator.Changed():
			updated, err := coreAccumulator.FilteredUpdate(ctx, &coreSnapshot, &[]*kates.Delta{}, isValid)
			if err != nil {
				return err
			}
			if !updated {
				continue
			}
			a.coreStore = NewCoreStore(&coreSnapshot)
		case callback, ok := <-rolloutCallback:
			if ok {
				dlog.Debugf(ctx, "argo rollout callback: %v", callback.EventType)
				a.rolloutStore, err = rolloutStore.FromCallback(callback)
				if err != nil {
					dlog.Warnf(ctx, "Error processing rollout callback: %s", err)
				}
			}
		case callback, ok := <-applicationCallback:
			if ok {
				dlog.Debugf(ctx, "argo application callback: %v", callback.EventType)
				a.applicationStore, err = applicationStore.FromCallback(callback)
				if err != nil {
					dlog.Warnf(ctx, "Error processing application callback: %s", err)
				}
			}
		case directive := <-a.newDirective:
			a.directiveHandler.HandleDirective(ctx, a, directive)
		}
		// only ask ambassador for a snapshot if we're actually going to report it.
		// if reportRunning is true, that means we're still in the quiet period
		// after sending a report.
		if !a.reportingStopped && !a.reportRunning.Value() {
			snapshot, err := getAmbSnapshotInfo(snapshotURL)
			if err != nil {
				dlog.Warnf(ctx, "Error getting snapshot from ambassador %+v", err)
			}
			dlog.Debug(ctx, "Received snapshot in agent")
			if err = a.ProcessSnapshot(ctx, snapshot, ambHost); err != nil {
				dlog.Warnf(ctx, "error processing snapshot: %+v", err)
			}
		}

		a.MaybeReport(ctx)
	}

}

func (a *Agent) MaybeReport(ctx context.Context) {
	if a.ambassadorAPIKey == "" {
		dlog.Debugf(ctx, "CLOUD_CONNECT_TOKEN not set in the environment, not reporting snapshot")
		return
	}
	if a.reportingStopped || a.reportRunning.Value() || (a.reportToSend == nil) {
		// Don't report if the Director told us to stop reporting, if we are
		// already sending a report or waiting for the minimum time between
		// reports, or if there is nothing new to report right now.
		dlog.Debugf(ctx, "Not reporting snapshot [reporting stopped = %t] [report running = %t] [report to send is nil = %t]", a.reportingStopped, a.reportRunning.Value(), (a.reportToSend == nil))
		return
	}

	// It's time to send a report
	if a.comm == nil {
		// The communications channel to the DCP was not yet created or was
		// closed above, due to a change in identity, or close elsewhere, due to
		// a change in endpoint configuration.
		newComm, err := NewComm(ctx, a.connInfo, a.agentID, a.ambassadorAPIKey)
		if err != nil {
			dlog.Warnf(ctx, "Failed to dial the DCP: %v", err)
			dlog.Warn(ctx, "DCP functionality disabled until next retry")

			return
		}

		a.comm = newComm
		a.newDirective = a.comm.Directives()
	}
	a.reportRunning.Set(true) // Cleared when the report completes

	// Send a report. This is an RPC, i.e. it can block, so we do this in a
	// goroutine. Sleep after send so we don't need to keep track of
	// whether/when it's okay to send the next report.
	go func(ctx context.Context, report *agent.Snapshot, delay time.Duration) {
		var err error
		defer func() {
			if err != nil {
				dlog.Warnf(ctx, "failed to report: %+v", err)
			}
			dlog.Debugf(ctx, "Finished sending snapshot report, sleeping for %s", delay.String())
			time.Sleep(delay)
			a.reportRunning.Set(false)
			// make the write non blocking
			select {
			case a.reportComplete <- err:
				// cool we sent something
			default:
				// do nothing if nobody is listening
			}
		}()
		a.ambassadorAPIKeyMutex.Lock()
		apikey := a.ambassadorAPIKey
		a.ambassadorAPIKeyMutex.Unlock()
		err = a.comm.Report(ctx, report, apikey)

	}(ctx, a.reportToSend, a.minReportPeriod)

	// Update state variables
	a.reportToSend = nil // Set when a snapshot yields a fresh report
}

// ProcessSnapshot turns a Watt/Diag Snapshot into a report that the agent can
// send to the Director. If the new report is semantically different from the
// prior one sent, then the Agent's state is updated to indicate that reporting
// should occur once again.
func (a *Agent) ProcessSnapshot(ctx context.Context, snapshot *snapshotTypes.Snapshot, ambHost string) error {
	if snapshot == nil || snapshot.AmbassadorMeta == nil {
		dlog.Warn(ctx, "No metadata discovered for snapshot, not reporting.")
		return nil
	}

	agentID := GetIdentity(snapshot.AmbassadorMeta, ambHost)
	if agentID == nil {
		dlog.Warnf(ctx, "Could not parse identity info out of snapshot, not sending snapshot")
		return nil
	}
	a.agentID = agentID

	newConnInfo, err := connInfoFromAddress(a.connAddress)
	if err != nil {
		// The user has attempted to turn on the Agent (otherwise GetIdentity
		// would have returned nil), but there's a problem with the connection
		// configuration. Rather than processing the entire snapshot and then
		// failing to send the resulting report, let's just fail now. The user
		// will see the error in the logs and correct the configuration.
		return err
	}

	if a.connInfo == nil || *newConnInfo != *a.connInfo {
		// The configuration for the Director endpoint has changed: either this
		// is the first snapshot or the user changed the value.
		//
		// Close any existing communications channel so that we can create
		// a new one with the new endpoint.
		a.ClearComm()

		// Save the new endpoint information.
		a.connInfo = newConnInfo
	}

	if snapshot.Kubernetes != nil {
		if a.coreStore != nil {
			if a.coreStore.podStore != nil {
				snapshot.Kubernetes.Pods = a.coreStore.podStore.StateOfWorld()
				dlog.Debugf(ctx, "Found %d pods", len(snapshot.Kubernetes.Pods))
			}
			if a.coreStore.configMapStore != nil {
				snapshot.Kubernetes.ConfigMaps = a.coreStore.configMapStore.StateOfWorld()
				dlog.Debugf(ctx, "Found %d configMaps", len(snapshot.Kubernetes.ConfigMaps))
			}
			if a.coreStore.deploymentStore != nil {
				snapshot.Kubernetes.Deployments = a.coreStore.deploymentStore.StateOfWorld()
				dlog.Debugf(ctx, "Found %d Deployments", len(snapshot.Kubernetes.Deployments))
			}
			if a.coreStore.endpointStore != nil {
				snapshot.Kubernetes.Endpoints = a.coreStore.endpointStore.StateOfWorld()
				dlog.Debugf(ctx, "Found %d Endpoints", len(snapshot.Kubernetes.Endpoints))
			}
		}
		if a.rolloutStore != nil {
			snapshot.Kubernetes.ArgoRollouts = a.rolloutStore.StateOfWorld()
			dlog.Debugf(ctx, "Found %d argo rollouts", len(snapshot.Kubernetes.ArgoRollouts))
		}
		if a.applicationStore != nil {
			snapshot.Kubernetes.ArgoApplications = a.applicationStore.StateOfWorld()
			dlog.Debugf(ctx, "Found %d argo applications", len(snapshot.Kubernetes.ArgoApplications))
		}
		if a.apiDocsStore != nil {
			a.apiDocsStore.ProcessSnapshot(ctx, snapshot)
			snapshot.APIDocs = a.apiDocsStore.StateOfWorld()
			dlog.Debugf(ctx, "Found %d api docs", len(snapshot.APIDocs))
		}
	}

	if err = snapshot.Sanitize(); err != nil {
		return err
	}
	rawJsonSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	report := &agent.Snapshot{
		Identity:    agentID,
		RawSnapshot: rawJsonSnapshot,
		ContentType: snapshotTypes.ContentTypeJSON,
		ApiVersion:  snapshotTypes.ApiVersion,
		SnapshotTs:  timestamppb.Now(),
	}

	a.reportToSend = report

	return nil
}

var allowedMetricsSuffixes = []string{"upstream_rq_total", "upstream_rq_time", "upstream_rq_5xx"}

func (a *Agent) MetricsRelayHandler(logCtx context.Context, in *envoyMetrics.StreamMetricsMessage) {
	metrics := in.GetEnvoyMetrics()
	dlog.Debugf(logCtx, "received %d metrics", len(metrics))
	if a.comm != nil && !a.reportingStopped {
		a.ambassadorAPIKeyMutex.Lock()
		apikey := a.ambassadorAPIKey
		a.ambassadorAPIKeyMutex.Unlock()

		outMetrics := make([]*io_prometheus_client.MetricFamily, 0, len(metrics))
		for _, metricFamily := range metrics {
			for _, suffix := range allowedMetricsSuffixes {
				if strings.HasSuffix(metricFamily.GetName(), suffix) {
					outMetrics = append(outMetrics, metricFamily)
					break
				}
			}
		}

		outMessage := &agent.StreamMetricsMessage{
			Identity:     a.agentID,
			EnvoyMetrics: in.EnvoyMetrics,
		}
		dlog.Debugf(logCtx, "relaying %d metrics", len(outMessage.GetEnvoyMetrics()))
		if err := a.comm.StreamMetrics(logCtx, outMessage, apikey); err != nil {
			dlog.Errorf(logCtx, "Error streaming metrics: %+v", err)
		}
	}
}

// ClearComm ends the current connection to the Director, if it exists, thereby
// forcing a new connection to be created when needed.
func (a *Agent) ClearComm() {
	if a.comm != nil {
		a.comm.Close()
		a.comm = nil
	}
}

// MaxDuration returns the greater of two durations.
func MaxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
