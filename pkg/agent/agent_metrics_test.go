package agent

import (
	"context"
	"github.com/datawire/ambassador/v2/pkg/api/agent"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v3"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

var (
	counterType    = io_prometheus_client.MetricType_COUNTER
	acceptedMetric = &io_prometheus_client.MetricFamily{
		Name: StrToPointer("cluster.apple_prod_443.upstream_rq_total"),
		Type: &counterType,
		Metric: []*io_prometheus_client.Metric{
			{
				Counter: &io_prometheus_client.Counter{
					Value: Float64ToPointer(42),
				},
				TimestampMs: Int64ToPointer(time.Now().Unix() * 1000),
			},
		},
	}
	ignoredMetric = &io_prometheus_client.MetricFamily{
		Name: StrToPointer("cluster.apple_prod_443.metric_to_ignore"),
		Type: &counterType,
		Metric: []*io_prometheus_client.Metric{
			{
				Counter: &io_prometheus_client.Counter{
					Value: Float64ToPointer(42),
				},
				TimestampMs: Int64ToPointer(time.Now().Unix() * 1000),
			},
		},
	}
)

type AgentMetricsSuite struct {
	suite.Suite

	clientMock *MockClient

	stubbedAgent *Agent
}

func (s *AgentMetricsSuite) SetupTest() {
	s.clientMock = &MockClient{}

	s.stubbedAgent = &Agent{
		metricsRelayDeadline: time.Time{},
		comm: &RPCComm{
			client: s.clientMock,
		},
	}
}

func (s *AgentMetricsSuite) AfterTest(suiteName, testName string) {
	return
}

func (s *AgentMetricsSuite) TestMetricsHandlerWithRelay() {
	//given
	ctx := context.TODO()

	//when
	s.stubbedAgent.MetricsRelayHandler(ctx, &envoyMetrics.StreamMetricsMessage{
		Identifier:   nil,
		EnvoyMetrics: []*io_prometheus_client.MetricFamily{ignoredMetric, acceptedMetric},
	})

	//then
	assert.Equal(s.T(), []*agent.StreamMetricsMessage{{
		EnvoyMetrics: []*io_prometheus_client.MetricFamily{acceptedMetric},
	}}, s.clientMock.SentMetrics)
}

func (s *AgentMetricsSuite) TestMetricsHandlerWithRelayPass() {
	//given
	ctx := context.TODO()
	s.stubbedAgent.metricsRelayDeadline = time.Now().Add(defaultMinReportPeriod)

	//when
	s.stubbedAgent.MetricsRelayHandler(ctx, &envoyMetrics.StreamMetricsMessage{
		Identifier:   nil,
		EnvoyMetrics: []*io_prometheus_client.MetricFamily{acceptedMetric},
	})

	//then
	assert.Equal(s.T(), 0, len(s.clientMock.SentMetrics))
}

func TestSuiteAgentMetrics(t *testing.T) {
	suite.Run(t, new(AgentMetricsSuite))
}
