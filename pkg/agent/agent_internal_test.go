package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"google.golang.org/protobuf/types/known/durationpb"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/datawire/ambassador/pkg/api/agent"
	"github.com/datawire/ambassador/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// Take a json formatted string and transform it to kates.Unstructured
// for easy formatting of Snapshot.Invalid members
func getUnstructured(objStr string) *kates.Unstructured {
	var obj map[string]interface{}
	_ = json.Unmarshal([]byte(objStr), &obj)
	unstructured := &kates.Unstructured{}
	unstructured.SetUnstructuredContent(obj)
	return unstructured
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func getRandomAmbassadorID() string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// gets a context for logging and canceling
func getCtxLog() (context.Context, context.CancelFunc) {
	llog := logrus.New()
	llog.SetLevel(logrus.DebugLevel)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = dlog.WithLogger(ctx, dlog.WrapLogrus(llog))

	return ctx, cancel
}

func TestHandleAPIKeyConfigChange(t *testing.T) {
	t.Parallel()
	objMeta := metav1.ObjectMeta{
		Name:      "coolname",
		Namespace: "coolnamespace",
	}
	testcases := []struct {
		testName       string
		agent          *Agent
		secrets        []kates.Secret
		configMaps     []kates.ConfigMap
		expectedAPIKey string
	}{
		{
			testName: "configmap-wins",
			agent: &Agent{
				agentNamespace:               "coolnamespace",
				agentCloudResourceConfigName: "coolname",
				ambassadorAPIKey:             "",
				ambassadorAPIKeyEnvVarValue:  "",
			},
			secrets: []kates.Secret{},
			configMaps: []kates.ConfigMap{
				{
					ObjectMeta: objMeta,
					Data: map[string]string{
						"CLOUD_CONNECT_TOKEN": "beepboop",
					},
				},
			},
			expectedAPIKey: "beepboop",
		},
		{
			testName: "secret-over-configmap",
			agent: &Agent{
				agentNamespace:               "coolnamespace",
				agentCloudResourceConfigName: "coolname",
				ambassadorAPIKey:             "",
				ambassadorAPIKeyEnvVarValue:  "",
			},
			secrets: []kates.Secret{
				{
					ObjectMeta: objMeta,
					Data: map[string][]byte{
						"CLOUD_CONNECT_TOKEN": []byte("secretvalue"),
					},
				},
			},
			configMaps: []kates.ConfigMap{
				{
					ObjectMeta: objMeta,
					Data: map[string]string{
						"CLOUD_CONNECT_TOKEN": "beepboop",
					},
				},
			},
			expectedAPIKey: "secretvalue",
		},
		{
			testName: "from-secret",
			agent: &Agent{
				agentNamespace:               "coolnamespace",
				agentCloudResourceConfigName: "coolname",
				ambassadorAPIKey:             "",
				ambassadorAPIKeyEnvVarValue:  "",
			},
			secrets: []kates.Secret{
				{
					ObjectMeta: objMeta,
					Data: map[string][]byte{
						"CLOUD_CONNECT_TOKEN": []byte("secretvalue"),
					},
				},
			},
			configMaps:     []kates.ConfigMap{},
			expectedAPIKey: "secretvalue",
		},
		{
			testName: "configmap-empty-string-value",
			agent: &Agent{
				agentNamespace:               "coolnamespace",
				agentCloudResourceConfigName: "coolname",
				ambassadorAPIKey:             "someexistingvalue",
				ambassadorAPIKeyEnvVarValue:  "",
			},
			secrets: []kates.Secret{},
			configMaps: []kates.ConfigMap{
				{
					ObjectMeta: objMeta,
					Data:       map[string]string{},
				},
			},
			expectedAPIKey: "",
		},
		{
			testName: "secret-empty-string-value",
			agent: &Agent{
				agentNamespace:               "coolnamespace",
				agentCloudResourceConfigName: "coolname",
				ambassadorAPIKey:             "someexistingvalue",
				ambassadorAPIKeyEnvVarValue:  "",
			},
			secrets: []kates.Secret{
				{
					ObjectMeta: objMeta,
					Data:       map[string][]byte{},
				},
			},
			configMaps:     []kates.ConfigMap{},
			expectedAPIKey: "",
		},
		{
			testName: "fall-back-envvar",
			agent: &Agent{
				agentNamespace:               "coolnamespace",
				agentCloudResourceConfigName: "coolname",
				ambassadorAPIKey:             "somevaluefromsomewhereelse",
				ambassadorAPIKeyEnvVarValue:  "gotfromenv",
			},
			expectedAPIKey: "gotfromenv",
		},
		{
			testName: "fall-back-envvar-bad-configs",
			agent: &Agent{
				agentNamespace:               "notcoolnamespace",
				agentCloudResourceConfigName: "notcoolname",
				ambassadorAPIKey:             "somevaluefromsomewhereelse",
				ambassadorAPIKeyEnvVarValue:  "gotfromenv",
			},
			secrets: []kates.Secret{
				{
					ObjectMeta: objMeta,
					Data: map[string][]byte{
						"CLOUD_CONNECT_TOKEN": []byte("secretvalue"),
					},
				},
			},
			configMaps: []kates.ConfigMap{
				{
					ObjectMeta: objMeta,
					Data: map[string]string{
						"CLOUD_CONNECT_TOKEN": "secretvalue",
					},
				},
			},
			expectedAPIKey: "gotfromenv",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.testName, func(innerT *testing.T) {
			ctx, _ := getCtxLog()

			tc.agent.handleAPIKeyConfigChange(ctx, tc.secrets, tc.configMaps)

			assert.Equal(t, tc.agent.ambassadorAPIKey, tc.expectedAPIKey)

		})
	}
}

func TestProcessSnapshot(t *testing.T) {
	t.Parallel()
	snapshotTests := []struct {
		// name of test (passed to t.Run())
		testName string
		// snapshot to call ProcessSnapshot with
		inputSnap *snapshotTypes.Snapshot
		// expected return value of ProcessSnapshot
		ret error
		// expected value of inputSnap.reportToSend after calling ProcessSnapshot
		res *agent.Snapshot
		// expected value of Agent.connInfo after calling ProcessSnapshot
		// in certain circumstances, ProcessSnapshot resets that info
		expectedConnInfo *ConnInfo
		podStore         *podStore
		assertionFunc    func(*testing.T, *agent.Snapshot)
		address          string
	}{
		{
			// Totally nil inputs should not error and not panic, and should not set
			// snapshot.reportToSend
			testName:  "nil-snapshot",
			inputSnap: nil,
			ret:       nil,
			res:       nil,
		},
		{
			// If no ambassador modules exist in the snapshot, we should not try to send
			// a report.
			// More granular tests for this are in report_test.go
			testName: "no-modules",
			inputSnap: &snapshotTypes.Snapshot{
				AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{},
				Kubernetes:     &snapshotTypes.KubernetesSnapshot{},
			},
			ret: nil,
			res: nil,
		},
		{
			// if we let address be an empty string, the defaults should get set
			testName: "default-connection-info",
			inputSnap: &snapshotTypes.Snapshot{
				AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{
					AmbassadorID:      "default",
					ClusterID:         "dopecluster",
					AmbassadorVersion: "v1.0",
				},
				Kubernetes: &snapshotTypes.KubernetesSnapshot{},
			},
			// should not error
			ret: nil,
			res: &agent.Snapshot{
				Identity: &agent.Identity{
					Version:   "",
					Hostname:  "ambassador-host",
					License:   "",
					ClusterId: "dopecluster",
					Label:     "",
				},
				ContentType: snapshotTypes.ContentTypeJSON,
				ApiVersion:  snapshotTypes.ApiVersion,
			},
			expectedConnInfo: &ConnInfo{hostname: "app.getambassador.io", port: "443", secure: true},
		},
		{
			// ProcessSnapshot should set the Agent.connInfo to the parsed url from the
			// ambassador module's CEPC config
			testName: "module-contains-connection-info",
			address:  "http://somecooladdress:1234",
			inputSnap: &snapshotTypes.Snapshot{
				AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{
					AmbassadorID:      "default",
					AmbassadorVersion: "v1.1",
					ClusterID:         "clusterid",
				},
				Kubernetes: &snapshotTypes.KubernetesSnapshot{},
			},
			ret: nil,
			res: &agent.Snapshot{
				Identity: &agent.Identity{
					Version:   "",
					Hostname:  "ambassador-host",
					License:   "",
					ClusterId: "clusterid",
					Label:     "",
				},
				ContentType: snapshotTypes.ContentTypeJSON,
				ApiVersion:  snapshotTypes.ApiVersion,
			},
			// this matches what's in
			// `address`
			expectedConnInfo: &ConnInfo{hostname: "somecooladdress", port: "1234", secure: false},
		},
		{
			// if the agent has pods that match the service selector labels, it should
			// return those pods in the snapshot
			testName: "pods-in-snapshot",
			inputSnap: &snapshotTypes.Snapshot{
				AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{
					AmbassadorID:      "default",
					ClusterID:         "dopecluster",
					AmbassadorVersion: "v1.0",
				},
				Kubernetes: &snapshotTypes.KubernetesSnapshot{
					Services: []*kates.Service{
						{
							Spec: kates.ServiceSpec{
								Selector: map[string]string{"label": "matching"},
							},
						},
						{
							Spec: kates.ServiceSpec{
								Selector: map[string]string{"label2": "alsomatching", "label3": "yay"},
							},
						},
					},
				},
			},
			podStore: NewPodStore([]*kates.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns",
						Labels:    map[string]string{"label": "matching", "tag": "1.0"},
					},
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns",
						Labels:    map[string]string{"label2": "alsomatching", "tag": "1.0", "label3": "yay"},
					},
					Status: v1.PodStatus{
						Phase: v1.PodFailed,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod3",
						Namespace: "ns",
						Labels:    map[string]string{"label2": "alsomatching", "tag": "1.0"},
					},
					Status: v1.PodStatus{
						Phase: v1.PodSucceeded,
					},
				},
			}),
			// should not error
			ret: nil,
			res: &agent.Snapshot{
				Identity: &agent.Identity{
					Version:   "",
					Hostname:  "ambassador-host",
					License:   "",
					ClusterId: "dopecluster",
					Label:     "",
				},
				ContentType: snapshotTypes.ContentTypeJSON,
				ApiVersion:  snapshotTypes.ApiVersion,
			},
			expectedConnInfo: &ConnInfo{hostname: "app.getambassador.io", port: "443", secure: true},
			assertionFunc: func(t1 *testing.T, agentSnap *agent.Snapshot) {
				assert.NotEmpty(t1, agentSnap.RawSnapshot)
				ambSnap := &snapshotTypes.Snapshot{}
				err := json.Unmarshal(agentSnap.RawSnapshot, ambSnap)
				assert.Nil(t1, err)
				assert.Equal(t1, len(ambSnap.Kubernetes.Services), 2)
				assert.Equal(t1, len(ambSnap.Kubernetes.Pods), 2)
				for _, p := range ambSnap.Kubernetes.Pods {
					assert.Contains(t1, []string{"pod1", "pod2"}, p.ObjectMeta.Name)
				}
			},
		},
	}

	for _, testcase := range snapshotTests {
		t.Run(testcase.testName, func(innerT *testing.T) {
			a := NewAgent(nil)
			ctx, _ := getCtxLog()
			a.coreStore = &coreStore{podStore: testcase.podStore}
			a.connAddress = testcase.address

			actualRet := a.ProcessSnapshot(ctx, testcase.inputSnap, "ambassador-host")

			assert.Equal(innerT, testcase.ret, actualRet)
			if testcase.res == nil {
				assert.Nil(innerT, a.reportToSend)
			} else {
				assert.NotNil(innerT, a.reportToSend)
				assert.Equal(innerT, testcase.res.Identity, a.reportToSend.Identity)
				assert.Equal(innerT, testcase.res.ContentType, a.reportToSend.ContentType)
				assert.Equal(innerT, testcase.res.ApiVersion, a.reportToSend.ApiVersion)
			}
			if testcase.expectedConnInfo != nil {
				assert.Equal(innerT, testcase.expectedConnInfo, a.connInfo)
			}
			if testcase.assertionFunc != nil {
				testcase.assertionFunc(innerT, a.reportToSend)
			}
		})
	}
}

type mockAccumulator struct {
	changedChan     chan struct{}
	targetInterface interface{}
}

func (m *mockAccumulator) Changed() chan struct{} {
	return m.changedChan
}

func (m *mockAccumulator) FilteredUpdate(target interface{}, deltas *[]*kates.Delta, predicate func(*kates.Unstructured) bool) bool {
	rawtarget, err := json.Marshal(m.targetInterface)

	if err != nil {
		return false
	}
	err = json.Unmarshal(rawtarget, target)
	return true
}

// Set up a watch and send a MinReportPeriod directive to the directive channel
// Make sure that Agent.MinReportPeriod is set to this new value
func TestWatchReportPeriodDirective(t *testing.T) {
	t.Parallel()
	ctx, cancel := getCtxLog()

	a := NewAgent(nil)
	watchDone := make(chan error)

	directiveChan := make(chan *agent.Directive)
	a.newDirective = directiveChan
	cfgDuration, err := time.ParseDuration("1ms")
	assert.Nil(t, err)
	// initial report period is 1 second
	a.minReportPeriod = cfgDuration
	// we expect it to be set to 5 seconds
	expectedDuration, err := time.ParseDuration("50s10ns")
	assert.Nil(t, err)

	podAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	configAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	rolloutCallback := make(chan *GenericCallback)
	appCallback := make(chan *GenericCallback)

	go func() {
		err := a.watch(ctx, "http://localhost:9697", configAcc, podAcc, rolloutCallback, appCallback)
		watchDone <- err
	}()
	dur := durationpb.Duration{
		Seconds: 50,
		Nanos:   10,
	}

	// send directive thru the channel
	directive := &agent.Directive{
		ID:              "myid123",
		MinReportPeriod: &dur,
	}
	directiveChan <- directive

	// since we're async let's just sleep for a sec
	time.Sleep(1)

	// stop the watch
	cancel()

	select {
	case err := <-watchDone:
		assert.Nil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for watch to finish after cancelling context")
	}
	// make sure that the agent's min report period is what we expect
	assert.Equal(t, expectedDuration, a.minReportPeriod)
	assert.False(t, a.reportRunning.Value())
}

// Start a watch and send a nil then empty directive through the channel
// make sure nothing errors or panics
func TestWatchEmptyDirectives(t *testing.T) {
	t.Parallel()
	ctx, cancel := getCtxLog()

	a := NewAgent(nil)
	id := agent.Identity{}
	a.agentID = &id
	watchDone := make(chan error)
	directiveChan := make(chan *agent.Directive)
	a.newDirective = directiveChan

	podAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	configAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	rolloutCallback := make(chan *GenericCallback)
	appCallback := make(chan *GenericCallback)
	go func() {
		err := a.watch(ctx, "http://localhost:9697", configAcc, podAcc, rolloutCallback, appCallback)
		watchDone <- err
	}()

	// sending a direcitve with nothing set should not error
	directive := &agent.Directive{}
	directiveChan <- directive
	select {
	case err := <-watchDone:
		eString := "No error"
		if err != nil {
			eString = err.Error()
		}
		t.Fatalf("Sending empty directive stopped the watch and shouldn't have. Error: %s", eString)
	case <-time.After(2 * time.Second):
	}

	// sending nil also shouldn't crash things
	directiveChan <- nil
	select {
	case err := <-watchDone:
		eString := "No error"
		if err != nil {
			eString = err.Error()
		}
		t.Fatalf("Sending empty directive stopped the watch and shouldn't have. Error: %s", eString)
	case <-time.After(2 * time.Second):
	}

	cancel()

	select {
	case err := <-watchDone:
		assert.Nil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for watch to finish after cancelling context")
	}
}

// Setup a watch
// send a directive to tell the agent to stop sending reports to the agent comm.
// Then, send a snapshot through the channel and ensure that it doesn't get sent to the agent com
func TestWatchStopReportingDirective(t *testing.T) {
	t.Parallel()
	ctx, cancel := getCtxLog()

	a := NewAgent(nil)
	id := agent.Identity{}
	a.agentID = &id
	watchDone := make(chan error)
	directiveChan := make(chan *agent.Directive)
	a.newDirective = directiveChan

	// setup our mock client
	client := &MockClient{}
	c := &RPCComm{
		conn:       client,
		client:     client,
		rptWake:    make(chan struct{}, 1),
		retCancel:  cancel,
		agentID:    &id,
		directives: directiveChan,
	}
	a.comm = c
	a.connInfo = &ConnInfo{hostname: "localhost", port: "8080", secure: false}
	podAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	configAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	rolloutCallback := make(chan *GenericCallback)
	appCallback := make(chan *GenericCallback)

	// start watch
	go func() {
		err := a.watch(ctx, "http://localhost:9697", configAcc, podAcc, rolloutCallback, appCallback)
		watchDone <- err
	}()

	// send directive to stop reporting
	directive := &agent.Directive{
		ID:            "1234",
		StopReporting: true,
	}
	directiveChan <- directive
	// since we're async just wait a sec
	time.Sleep(time.Second * 3)

	// cancel the watch
	cancel()

	select {
	case err := <-watchDone:
		assert.Nil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for watch to finish after cancelling context")
	}
	// make sure that reportingStopped is still set
	assert.True(t, a.reportingStopped)
	// assert that no snapshots were sent
	assert.Equal(t, len(client.GetSnapshots()), 0, "No snapshots should have been sent to the client")
	assert.False(t, a.reportRunning.Value())
}

// Start a watch. Configure the mock client to error when Report() is called
// Send a snapshot through the channel, and make sure the error propogates thru the agent.reportComplete
// channel, and that the error doesn't make things sad.
func TestWatchErrorSendingSnapshot(t *testing.T) {
	t.Parallel()
	ctx, cancel := getCtxLog()
	ambId := getRandomAmbassadorID()
	a := NewAgent(nil)
	a.reportingStopped = false
	a.reportRunning.Set(false)
	// set to 3 seconds so we can reliably assert that reportRunning is true later
	minReport, err := time.ParseDuration("3s")
	assert.Nil(t, err)
	a.minReportPeriod = minReport
	id := agent.Identity{}
	a.agentID = &id
	a.ambassadorAPIKey = "mycoolapikey"
	a.ambassadorAPIKeyEnvVarValue = a.ambassadorAPIKey
	a.agentCloudResourceConfigName = "bogusvalue"
	// needs to match `address` from moduleConfigRaw below
	a.connAddress = "http://localhost:8080"
	a.connInfo = &ConnInfo{hostname: "localhost", port: "8080", secure: false}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// setup the snapshot we'll send
		snapshot := snapshotTypes.Snapshot{
			AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{
				AmbassadorID: ambId,
				ClusterID:    "reallylongthing",
			},
			Kubernetes: &snapshotTypes.KubernetesSnapshot{},
		}
		enSnapshot, err := json.Marshal(&snapshot)
		if err != nil {
			t.Fatal("error marshalling snapshot")
		}
		w.Write(enSnapshot)
	}))
	defer ts.Close()
	expectedErrorMessage := "Error sending report"

	client := &MockClient{
		// force an error
		reportFunc: func(ctx context.Context, in *agent.Snapshot) (*agent.SnapshotResponse, error) {
			return nil, errors.New(expectedErrorMessage)
		},
	}
	c := &RPCComm{
		conn:       client,
		client:     client,
		rptWake:    make(chan struct{}, 1),
		retCancel:  cancel,
		agentID:    &id,
		directives: make(chan *agent.Directive, 1),
	}
	a.comm = c

	watchDone := make(chan error)
	podAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	configAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	rolloutCallback := make(chan *GenericCallback)
	appCallback := make(chan *GenericCallback)

	// start the watch
	go func() {
		err := a.watch(ctx, ts.URL, configAcc, podAcc, rolloutCallback, appCallback)
		watchDone <- err
	}()

	// assert that report completes
	select {
	case err := <-a.reportComplete:
		// make sure that we got an error and that error is the same one we configured the
		// mock client to send
		assert.NotNil(t, err)
		assert.Equal(t, expectedErrorMessage, err.Error())
		assert.False(t, a.reportRunning.Value())
		cancel()
	case err := <-watchDone:
		if err != nil {
			t.Fatalf("Watch ended early with error %s", err.Error())
		} else {
			t.Fatal("Watch ended early with no error.")
		}
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("Timed out waiting for report to complete.")
	}
	select {
	case err := <-watchDone:
		assert.Nil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for watch to end")
	}
}

// Start a watch. Setup a mock client to capture what we would have sent to the agent com
// Send a snapshot with some data in it thru the channel
// Make sure the Snapshot.KubernetesSecrets and Snapshot.Invalid get scrubbed of sensitive data and
// we send a SnapshotTs that makes sense (so the agent com can throw out older snapshots)
func TestWatchWithSnapshot(t *testing.T) {
	t.Parallel()
	ctx, cancel := getCtxLog()
	clusterID := "coolcluster"
	ambId := getRandomAmbassadorID()
	a := NewAgent(nil)
	a.reportingStopped = false
	a.reportRunning.Set(false)

	id := agent.Identity{}
	// set to 0 seconds so we can reliably assert that report running is false later
	minReport, err := time.ParseDuration("0s")
	assert.Nil(t, err)
	a.minReportPeriod = minReport
	a.agentID = &id
	// needs to matched parsed ish below
	a.connAddress = "http://localhost:8080/"
	a.connInfo = &ConnInfo{hostname: "localhost", port: "8080", secure: false}
	apiKey := "coolapikey"
	a.ambassadorAPIKey = apiKey
	a.ambassadorAPIKeyEnvVarValue = apiKey
	a.agentCloudResourceConfigName = "bogusvalue"
	snapshot := &snapshotTypes.Snapshot{
		Invalid: []*kates.Unstructured{
			// everything that's not errors or metadata here needs to get scrubbed
			getUnstructured(`
{
"kind":"WeirdKind",
"apiVersion":"v1",
"metadata": {
"name":"hi",
"namespace":"default"
},
"errors": "someerrors",
"wat":"dontshowthis"
}`),
		},
		Kubernetes: &snapshotTypes.KubernetesSnapshot{
			Secrets: []*kates.Secret{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "ns",
						// make sure this gets unset
						Annotations: map[string]string{"also": "unset"},
					},
					Type: "Opaque",
					Data: map[string][]byte{
						// make sure these values get scrubbed
						"data1": []byte("d293YXNlY3JldA=="),
						"data2": []byte("d293YW5vdGhlcm9uZQ=="),
					},
				},
			},
		},
		AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{
			AmbassadorID:      ambId,
			ClusterID:         clusterID,
			AmbassadorVersion: "v1.0",
		},
	}
	// send a snapshot thru the channel
	// keep track of when we did that for assertions
	var snapshotSentTime time.Time
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enSnapshot, err := json.Marshal(&snapshot)
		if err != nil {
			t.Fatal("error marshalling snapshot")
		}
		w.Write(enSnapshot)
		snapshotSentTime = time.Now()
	}))
	defer ts.Close()

	// setup the mock client
	client := &MockClient{}
	c := &RPCComm{
		conn:       client,
		client:     client,
		rptWake:    make(chan struct{}, 1),
		retCancel:  cancel,
		agentID:    &id,
		directives: make(chan *agent.Directive, 1),
	}
	a.comm = c

	watchDone := make(chan error)
	podAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	configAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	rolloutCallback := make(chan *GenericCallback)
	appCallback := make(chan *GenericCallback)

	// start the watch
	go func() {
		err := a.watch(ctx, ts.URL, configAcc, podAcc, rolloutCallback, appCallback)
		watchDone <- err
	}()

	// assert that we send a couple of reports.
	// we just want to make sure we don't get stuck after sending one report
	// each report will be the same because the snapshot server we setup for this test is just
	// returning static content
	reportsSent := 0
	for reportsSent < 2 {
		select {
		case err := <-a.reportComplete:
			assert.Nil(t, err)
			reportsSent += 1
		case err := <-watchDone:
			t.Fatalf("Watch ended early with error %s", err.Error())
		case <-time.After(10 * time.Second):
			cancel()
			t.Fatal("Timed out waiting for report to complete.")
		}
	}
	cancel()

	// stop the watch and make sure if finishes without an error
	select {
	case err := <-watchDone:
		// make sure the watch finishes without a problem
		assert.Nil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for watch to finish after cancelling context")
	}
	sentSnaps := client.GetSnapshots()

	// Make sure that the client got a snapshot to send
	assert.NotNil(t, sentSnaps, "No snapshots sent")
	assert.GreaterOrEqual(t, len(sentSnaps), 1, "Should have sent at least 1 snapshot")
	lastMeta := client.GetLastMetadata()
	assert.NotNil(t, lastMeta)
	md := lastMeta.Get("x-ambassador-api-key")
	assert.NotEmpty(t, md)
	assert.Equal(t, md[0], apiKey)

	/////// Make sure the raw snapshot that got sent looks like we expect
	sentSnapshot := sentSnaps[0]
	var actualSnapshot snapshotTypes.Snapshot
	err = json.Unmarshal(sentSnapshot.RawSnapshot, &actualSnapshot)
	assert.Nil(t, err)

	// Assert invalid things got scrubbed
	assert.Equal(t, len(actualSnapshot.Invalid), 1)
	expectedInvalid := getUnstructured(`
{
"kind":"WeirdKind",
"apiVersion":"v1",
"metadata": {
"name":"hi",
"namespace":"default"
},
"errors":"someerrors"
}`)
	assert.Equal(t, actualSnapshot.Invalid[0], expectedInvalid)

	// make sure the secret values got scrubbed
	assert.NotNil(t, actualSnapshot.Kubernetes)
	assert.Equal(t, len(actualSnapshot.Kubernetes.Secrets), 1)
	assert.Equal(t, len(actualSnapshot.Kubernetes.Secrets[0].ObjectMeta.Annotations), 0)
	assert.Equal(t, "secret-1", actualSnapshot.Kubernetes.Secrets[0].Name)
	assert.Equal(t, "ns", actualSnapshot.Kubernetes.Secrets[0].Namespace)
	secretData := actualSnapshot.Kubernetes.Secrets[0].Data
	assert.NotEqual(t, []byte("d293YXNlY3JldA=="), secretData["data1"])
	assert.NotEqual(t, []byte("d293YW5vdGhlcm9uZQ=="), secretData["data2"])

	/////// Make sure that the timestamp we sent makes sense
	assert.NotNil(t, sentSnapshot.SnapshotTs)
	snapshotTime := sentSnapshot.SnapshotTs.AsTime()
	assert.WithinDuration(t, snapshotSentTime, snapshotTime, 5*time.Second)

	/////// assert API version and content type
	assert.Equal(t, snapshotTypes.ApiVersion, sentSnapshot.ApiVersion)
	assert.Equal(t, snapshotTypes.ContentTypeJSON, sentSnapshot.ContentType)

	/////// Identity assertions
	actualIdentity := sentSnapshot.Identity
	assert.NotNil(t, actualIdentity)
	assert.Equal(t, "", actualIdentity.AccountId)
	assert.NotNil(t, actualIdentity.Version)
	assert.Equal(t, "", actualIdentity.Version)
	assert.Equal(t, clusterID, actualIdentity.ClusterId)
	parsedURL, err := url.Parse(ts.URL)
	assert.Nil(t, err)
	assert.Equal(t, actualIdentity.Hostname, parsedURL.Hostname())
}

// Setup a watch.
// Send a snapshot with no cluster id
// Make sure we don't try to send anything and that nothing errors or panics
func TestWatchEmptySnapshot(t *testing.T) {
	t.Parallel()
	ctx, cancel := getCtxLog()

	a := NewAgent(nil)
	minReport, err := time.ParseDuration("1ms")
	assert.Nil(t, err)
	a.minReportPeriod = minReport
	watchDone := make(chan error)

	snapshotRequested := make(chan bool)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ambId := getRandomAmbassadorID()
		// setup the snapshot we'll send
		snapshot := snapshotTypes.Snapshot{
			AmbassadorMeta: &snapshotTypes.AmbassadorMetaInfo{
				AmbassadorID: ambId,
			},
		}
		enSnapshot, err := json.Marshal(&snapshot)
		if err != nil {
			t.Fatal("error marshalling snapshot")
		}

		w.Write(enSnapshot)
		select {
		case snapshotRequested <- true:
		default:
		}
	}))
	defer ts.Close()
	podAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	configAcc := &mockAccumulator{
		changedChan: make(chan struct{}),
	}
	rolloutCallback := make(chan *GenericCallback)
	appCallback := make(chan *GenericCallback)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				watchDone <- errors.New(fmt.Sprintf("%v", r))
				t.Errorf("Panic-ed while sending an empty snapshot")
			}
		}()
		err := a.watch(ctx, ts.URL, configAcc, podAcc, rolloutCallback, appCallback)
		watchDone <- err
	}()
	select {
	case <-snapshotRequested:
		cancel()
	case <-time.After(10 * time.Second):
		t.Fatalf("Timed out waiting for agent to request snapshot")
		cancel()
	}

	select {
	case err := <-watchDone:
		assert.Nil(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Watch did not end")
	}
	assert.False(t, a.reportRunning.Value())
}
