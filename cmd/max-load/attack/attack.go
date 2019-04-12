package attack

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/datawire/apro/cmd/max-load/metrics"
	vegeta "github.com/tsenart/vegeta/lib"
)

func maxFiles() int {
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		panic(err)
	}

	ret := rlimit.Cur

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Cur: rlimit.Max,
		Max: rlimit.Max,
	})
	if err == nil {
		ret = rlimit.Max
	}

	if ret > math.MaxInt32 {
		ret = math.MaxInt32
	}
	return int(ret)
}

var attacker = vegeta.NewAttacker(

	vegeta.TLSConfig(
		// #nosec G402
		&tls.Config{InsecureSkipVerify: true},
	),

	vegeta.Connections(maxFiles()), // setting -1 or 0 for no-limit doesn't seemt to work?
)

func SetHTTP2Enabled(enabled bool) {
	vegeta.HTTP2(enabled)(attacker)
}

var sourcePortRE = regexp.MustCompile(":[1-9][0-9]*->")

func openFiles() int {
	fis, _ := ioutil.ReadDir("/dev/fd/")
	return len(fis)
}

type TestCase struct {
	URL        string
	RPS        uint
	MaxLatency time.Duration

	ShouldStop func(metrics.MetricsReader) bool
}

type TestResult struct {
	Rate        uint
	Metrics     metrics.MetricsReader
	FilesBefore int
	FilesAfter  int
}

func (tc TestCase) Run() TestResult {
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    tc.URL,
		Header: http.Header(map[string][]string{"Authorization": {"Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ."}}),
	})

	m := metrics.NewMetrics()

	filesBefore := openFiles()
	for res := range attacker.Attack(targeter, vegeta.Rate{Freq: int(tc.RPS), Per: time.Second}, 0, fmt.Sprintf("atk-%d", tc.RPS)) {
		success := false
		limited := false
		switch res.Code {
		case http.StatusOK:
			success = true
		case http.StatusTooManyRequests:
			success = true
			limited = true
		}
		if success && tc.MaxLatency > 0 && res.Latency > tc.MaxLatency {
			success = false
			res.Error = "latency limit exceeded"
		}
		var errstr string
		if !success {
			errstr = fmt.Sprintf("code=%03d error=%#v x-envoy-overloaded=%#v body=%#v",
				res.Code,
				sourcePortRE.ReplaceAllString(res.Error, ":XYZ->"),
				res.Header.Get("x-envoy-overloaded"),
				string(res.Body),
			)
		}

		m.Add(success, limited, res.Latency, errstr)

		if tc.ShouldStop(m) {
			attacker.Stop()
		}
	}
	filesAfter := openFiles()

	return TestResult{
		Rate:        tc.RPS,
		Metrics:     m,
		FilesBefore: filesBefore,
		FilesAfter:  filesAfter,
	}
}

func (r TestResult) Passed(minSuccessRate float64) bool {
	return r.Metrics.SuccessRate() >= minSuccessRate
}

func (r TestResult) String(minSuccessRate float64) string {
	var prefix string
	if r.Passed(minSuccessRate) {
		prefix = "✔ Success"
	} else {
		prefix = "✘ Failed"
	}
	mainline := fmt.Sprintf("%s at rate=%drps (latency p95=%s max=%s margin(95%%)=±%s) (success rate=%d/%d=%f) (rate-limited: %d) (open files: %d→%d)",
		prefix,
		r.Rate,
		r.Metrics.LatencyQuantile(0.95), r.Metrics.LatencyMax(), r.Metrics.LatencyMargin(0.95),
		r.Metrics.CountSuccesses(), r.Metrics.CountRequests(), r.Metrics.SuccessRate(),
		r.Metrics.CountLimited(),
		r.FilesBefore, r.FilesAfter)
	errs := make([]string, 0, len(r.Metrics.Errors()))
	for err, n := range r.Metrics.Errors() {
		errs = append(errs, fmt.Sprintf("  error (%d): %s", n, err))
	}
	sort.Strings(errs)
	return strings.Join(append([]string{mainline}, errs...), "\n")
}
