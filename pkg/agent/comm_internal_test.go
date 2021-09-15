package agent

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/dlib/dlog"
)

type MockClient struct {
	Counter int64
	grpc.ClientStream
	SentSnapshots []*agent.Snapshot
	snapMux       sync.Mutex
	reportFunc    func(context.Context, *agent.Snapshot) (*agent.SnapshotResponse, error)
	LastMetadata  metadata.MD
}

func (m *MockClient) Close() error {
	return nil
}

func (m *MockClient) GetLastMetadata() metadata.MD {
	m.snapMux.Lock()
	defer m.snapMux.Unlock()
	meta := m.LastMetadata
	return meta
}

func (m *MockClient) GetSnapshots() []*agent.Snapshot {
	m.snapMux.Lock()
	defer m.snapMux.Unlock()
	snaps := m.SentSnapshots
	return snaps
}

func (m *MockClient) Report(ctx context.Context, in *agent.Snapshot, opts ...grpc.CallOption) (*agent.SnapshotResponse, error) {
	m.snapMux.Lock()
	defer m.snapMux.Unlock()
	if m.SentSnapshots == nil {
		m.SentSnapshots = []*agent.Snapshot{}
	}
	m.SentSnapshots = append(m.SentSnapshots, in)
	md, _ := metadata.FromOutgoingContext(ctx)
	m.LastMetadata = md
	if m.reportFunc != nil {
		return m.reportFunc(ctx, in)
	}
	return nil, nil
}

func (m *MockClient) Recv() (*agent.Directive, error) {
	counter := atomic.AddInt64(&m.Counter, 1)

	if counter < 3 {
		return &agent.Directive{
			Commands: []*agent.Command{
				{Message: fmt.Sprintf("test command %d", counter)},
			},
		}, nil
	}

	return nil, io.EOF
}

func (m *MockClient) Retrieve(ctx context.Context, in *agent.Identity, opts ...grpc.CallOption) (agent.Director_RetrieveClient, error) {
	fmt.Println("Retrieve called")
	return m, nil
}

type retrvsnapshotclient struct {
	grpc.ClientStream
}

func (r *retrvsnapshotclient) Recv() (*agent.RawSnapshotChunk, error) {
	return nil, nil
}

func (m *MockClient) RetrieveSnapshot(context.Context, *agent.Identity, ...grpc.CallOption) (agent.Director_RetrieveSnapshotClient, error) {
	return &retrvsnapshotclient{}, nil
}

func TestComm(t *testing.T) {
	llog := logrus.New()
	llog.SetLevel(logrus.DebugLevel)
	log := dlog.WrapLogrus(llog)
	ctx := dlog.WithLogger(context.Background(), log)
	ctx, cancel := context.WithCancel(ctx)
	client := &MockClient{}
	agentID := &agent.Identity{}
	c := &RPCComm{
		conn:       client,
		client:     client,
		rptWake:    make(chan struct{}, 1),
		retCancel:  cancel,
		agentID:    agentID,
		directives: make(chan *agent.Directive, 1),
	}

	go c.retrieveLoop(ctx)

	t.Logf("got: %v", <-c.directives)
	t.Logf("got: %v", <-c.directives)

	atomic.StoreInt64(&client.Counter, 0)

	if err := c.Report(ctx, &agent.Snapshot{
		Identity: agentID,
		Message:  "hello same ID",
	}, "apikey"); err != nil {
		t.Errorf("Comm.Report() error = %v", err)
	}

	t.Logf("got: %v", <-c.directives)
	t.Logf("got: %v", <-c.directives)

	eqID := &agent.Identity{}

	if err := c.Report(ctx, &agent.Snapshot{
		Identity: eqID,
		Message:  "hello equivalent ID",
	}, "apikey"); err != nil {
		t.Errorf("Comm.Report() error = %v", err)
	}

	if err := c.Close(); err != nil {
		t.Errorf("Comm.Close() error = %v", err)
	}
}

func TestConnInfo(t *testing.T) {
	assert := assert.New(t)

	var (
		ci  *ConnInfo
		err error
	)

	defaults := []string{
		"",
		fmt.Sprintf("https://%s:%s/", defaultHostname, defaultPort),
		"a bogus value that looks like a path",
	}

	for _, addr := range defaults {
		ci, err = connInfoFromAddress(addr)

		assert.NoError(err)
		assert.Equal(defaultHostname, ci.hostname)
		assert.Equal(defaultPort, ci.port)
		assert.True(ci.secure)
	}

	ci, err = connInfoFromAddress(":a bad value")
	assert.Error(err, ci)
}
