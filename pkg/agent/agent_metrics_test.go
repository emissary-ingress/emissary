package agent

import (
	"github.com/datawire/ambassador/v2/pkg/api/agent"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v3"
	"github.com/datawire/dlib/dlog"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"
	"net"
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

func agentMetricsSetupTest() (*MockClient, *Agent) {
	clientMock := &MockClient{}

	stubbedAgent := &Agent{
		metricsBackoffUntil: time.Time{},
		comm: &RPCComm{
			client: clientMock,
		},
		aggregatedMetrics: map[string][]*io_prometheus_client.MetricFamily{},
	}

	return clientMock, stubbedAgent
}

func TestMetricsRelayHandler(t *testing.T) {

	t.Run("will relay metrics from the stack", func(t *testing.T) {
		//given
		clientMock, stubbedAgent := agentMetricsSetupTest()
		ctx := peer.NewContext(dlog.NewTestContext(t, true), &peer.Peer{
			Addr: &net.IPAddr{
				IP: net.ParseIP("192.168.0.1"),
			},
		})
		stubbedAgent.aggregatedMetrics["192.168.0.1"] = []*io_prometheus_client.MetricFamily{acceptedMetric}

		//when
		stubbedAgent.MetricsRelayHandler(ctx, &envoyMetrics.StreamMetricsMessage{
			Identifier: nil,
			// ignored since time to report.
			EnvoyMetrics: []*io_prometheus_client.MetricFamily{ignoredMetric, acceptedMetric},
		})

		//then
		assert.Equal(t, []*agent.StreamMetricsMessage{{
			EnvoyMetrics: []*io_prometheus_client.MetricFamily{acceptedMetric},
		}}, clientMock.SentMetrics, "metrics should be propagated to cloud")
	})
	t.Run("will not relay the metrics since it is in cool down period.", func(t *testing.T) {
		//given
		clientMock, stubbedAgent := agentMetricsSetupTest()
		ctx := peer.NewContext(dlog.NewTestContext(t, true), &peer.Peer{
			Addr: &net.IPAddr{
				IP: net.ParseIP("192.168.0.1"),
			},
		})
		stubbedAgent.metricsBackoffUntil = time.Now().Add(defaultMinReportPeriod)

		//when
		stubbedAgent.MetricsRelayHandler(ctx, &envoyMetrics.StreamMetricsMessage{
			Identifier:   nil,
			EnvoyMetrics: []*io_prometheus_client.MetricFamily{acceptedMetric},
		})

		//then
		assert.Equal(t, stubbedAgent.aggregatedMetrics["192.168.0.1"],
			[]*io_prometheus_client.MetricFamily{acceptedMetric},
			"metrics should be added to the stack")
		assert.Equal(t, 0, len(clientMock.SentMetrics), "nothing send to cloud")
	})
	t.Run("peer IP is not available", func(t *testing.T) {
		// given
		clientMock, stubbedAgent := agentMetricsSetupTest()
		ctx := dlog.NewTestContext(t, true)

		//when
		stubbedAgent.MetricsRelayHandler(ctx, &envoyMetrics.StreamMetricsMessage{
			Identifier:   nil,
			EnvoyMetrics: []*io_prometheus_client.MetricFamily{acceptedMetric},
		})

		//then
		assert.Equal(t, 0, len(stubbedAgent.aggregatedMetrics), "no metrics")
		assert.Equal(t, 0, len(clientMock.SentMetrics), "nothing send to cloud")
	})
	t.Run("not metrics available in aggregatedMetrics", func(t *testing.T) {
		// given
		clientMock, stubbedAgent := agentMetricsSetupTest()
		ctx := peer.NewContext(dlog.NewTestContext(t, true), &peer.Peer{
			Addr: &net.IPAddr{
				IP: net.ParseIP("192.168.0.1"),
			},
		})

		//when
		stubbedAgent.MetricsRelayHandler(ctx, &envoyMetrics.StreamMetricsMessage{
			Identifier:   nil,
			EnvoyMetrics: []*io_prometheus_client.MetricFamily{},
		})

		//then
		assert.Equal(t, 0, len(stubbedAgent.aggregatedMetrics), "no metrics")
		assert.Equal(t, 0, len(clientMock.SentMetrics), "nothing send to cloud")
	})
}
