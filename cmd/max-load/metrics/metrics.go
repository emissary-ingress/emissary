package metrics

import (
	"math"
	"time"

	"github.com/aclements/go-moremath/stats"
	"github.com/influxdata/tdigest"
)

type MetricsReader interface {
	CountRequests() uint
	CountSuccesses() uint
	CountLimited() uint

	SuccessRate() float64

	LatencyQuantile(q float64) time.Duration
	LatencyMin() time.Duration
	LatencyMax() time.Duration
	LatencyMargin(confidence float64) time.Duration

	Errors() map[string]uint
}

type Metrics interface {
	MetricsReader
	Add(success bool, limited bool, latency time.Duration, err string)
}

type metrics struct {
	statsBasic     stats.StreamStats
	statsQuantile  *tdigest.TDigest
	statsSuccesses uint
	statsLimited   uint
	statsErrors    map[string]uint
}

func NewMetrics() Metrics {
	return &metrics{
		statsQuantile: tdigest.NewWithCompression(100), // mimics vegeta.LatencyMetrics
		statsErrors:   make(map[string]uint),
	}
}

func (m *metrics) Add(success bool, limited bool, latency time.Duration, errstr string) {
	if success {
		m.statsSuccesses++
	}
	if limited {
		m.statsLimited++
	}
	m.statsBasic.Add(float64(latency))
	m.statsQuantile.Add(float64(latency), 1)
	if errstr != "" {
		m.statsErrors[errstr] += 1
	}
}

func (m *metrics) CountRequests() uint     { return m.statsBasic.Count }
func (m *metrics) CountSuccesses() uint    { return m.statsSuccesses }
func (m *metrics) CountLimited() uint      { return m.statsLimited }
func (m *metrics) Errors() map[string]uint { return m.statsErrors }

func (m *metrics) SuccessRate() float64 {
	return float64(m.CountSuccesses()) / float64(m.CountRequests())
}

func (m *metrics) LatencyQuantile(q float64) time.Duration {
	return time.Duration(m.statsQuantile.Quantile(q))
}

func (m *metrics) LatencyMin() time.Duration { return time.Duration(m.statsBasic.Min) }
func (m *metrics) LatencyMax() time.Duration { return time.Duration(m.statsBasic.Max) }

func tCrit(df, n float64) float64 {
	return math.Abs(stats.InvCDF(stats.TDist{V: df})(n))
}

func (m *metrics) LatencyMargin(confidence float64) time.Duration {
	return time.Duration(tCrit(float64(m.statsBasic.Count-1), (1-confidence)/2) * m.statsBasic.StdDev() / math.Sqrt(float64(m.statsBasic.Count)))
}
