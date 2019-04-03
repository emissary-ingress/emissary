package main

import (
	"os/exec"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

var MaxIdleConnections = 3200

var attacker = vegeta.NewAttacker(
	vegeta.TLSConfig(&tls.Config{InsecureSkipVerify: true}), // #nosec G402
	vegeta.HTTP2(true),
	vegeta.Connections(MaxIdleConnections),
)

var sourcePortRE = regexp.MustCompile(":[1-9][0-9]*->")

func openFiles() int {
	fis, _ := ioutil.ReadDir("/dev/fd/")
	return len(fis)
}

type TestCase struct {
	URL string
	RPS int
	Duration time.Duration
}

type TestResult struct {
	Rate      int
	Successes uint64
	Requests  uint64
	Latency   time.Duration
	Errors    map[string]uint64
	FilesBefore     int
	FilesAfter     int
}

func RunTestRaw(tc TestCase) TestResult {
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    tc.URL,
		Header: http.Header(map[string][]string{"Authorization": {"Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ."}}),
	})
	vegetaRate := vegeta.Rate{Freq: tc.RPS, Per: time.Second}
	name := "atk-" + string(tc.RPS)
	var metrics vegeta.Metrics
	var successes uint64
	errs := make(map[string]uint64)
	filesBefore := openFiles()
	for res := range attacker.Attack(targeter, vegetaRate, tc.Duration, name) {
		// vegeta.Metrics doesn't consider HTTP 429 Too Many
		// Requests to be a "success", but for testing the
		// rate limit service, we should.
		switch res.Code {
		case http.StatusOK:
			successes++
		case http.StatusTooManyRequests:
			successes++
			res.Error = ""
		default:
		}
		if res.Error != "" {
			res.Error = sourcePortRE.ReplaceAllString(res.Error, ":XYZ->")
			n := errs[res.Error]
			errs[res.Error] = n + 1
		}
		metrics.Add(res)
	}
	metrics.Close()
	filesAfter := openFiles()

	return TestResult{tc.RPS, successes, metrics.Requests, metrics.Latencies.P95, errs, filesBefore, filesAfter}
}

func (r TestResult) SuccessRate() float64 {
	return float64(r.Successes) / float64(r.Requests)
}

func (r TestResult) Passed() bool {
	return r.Successes == r.Requests
}

func (r TestResult) String() string {
	var prefix string
	if r.Passed() {
		prefix = "✔ Success"
	} else {
		prefix = "✘ Failed"
	}
	mainline := fmt.Sprintf("%s at rate=%drps (latency %s) (success rate=%d/%d=%f) (open files: %d→%d)",
		prefix, r.Rate, r.Latency, r.Successes, r.Requests, r.SuccessRate(), r.FilesBefore, r.FilesAfter)
	errs := make([]string, 0, len(r.Errors))
	for err, n := range r.Errors {
		errs = append(errs, fmt.Sprintf("  error (%d): %s", n, err))
	}
	sort.Strings(errs)
	return strings.Join(append([]string{mainline}, errs...), "\n")
}

func RunTest(url string, rate int) bool {
	runs := 0
	for {
		result := RunTestRaw(TestCase{URL: url, RPS: rate, Duration: 5*time.Second})
		runs++
		fmt.Println(result)
		// Let the client cool down; let any dangling but not keep-alive connections die.
		cnt := openFiles()
		for cnt > MaxIdleConnections+200 {
			fmt.Printf("  cooldown: open files: %d\n", cnt)
			time.Sleep(time.Second)
			cnt = openFiles()
		}
		if result.Passed() {
			return true
		}
		// Let Ambassador cool down; require 1000 successful requests in a row.
		var passed uint64
		for passed < 1000 {
			// Use an RPS for which it is likely that `latency*rps < 1s`.  10ms latency
			// seems reasonable under non-load, so 100rps, but give it a little more
			// leeway at 75rps.
			cooldown := RunTestRaw(TestCase{URL: url, RPS: 75, Duration: time.Second})
			if cooldown.Passed() {
				passed += cooldown.Requests
			} else {
				passed = 0
			}
			fmt.Printf("  cooldown: %d (open files: %d→%d)\n", passed, cooldown.FilesBefore, cooldown.FilesAfter)
		}
		// try it up to 3 times
		if runs == 3 {
			return false
		}
	}
}

func usage() {
	fmt.Printf("Usage: %s [namespace] request_path\n", os.Args[0])
}

func parseArgs(args []string) string {
	var argNamespace string
	var argPath string
	switch len(args) {
	case 0:
		usage()
		os.Exit(2)
	case 1:
		argNamespace = "default"
		argPath = args[0]
	case 2:
		argNamespace = args[0]
		argPath = args[1]
	default:
		usage()
		os.Exit(2)
	}

	var nodeIP string
	bs, _ := exec.Command("kubectl", "config", "view", "--output=go-template", "--template={{range .clusters}}{{.cluster.server}}{{end}}").Output()
	for _, line := range strings.Split(string(bs), "\n") {
		parts := strings.Split(strings.TrimPrefix(line, "https://"), ":")
		nodeIP = parts[0]
	}

	bs, _ = exec.Command("kubectl", "--namespace="+argNamespace, "get", "service", "ambassador", "--output=go-template", "--template={{range .spec.ports}}{{if eq .name \"https\"}}{{.nodePort}}{{end}}{{end}}").Output()
	nodePort := strings.TrimSpace(string(bs))

	return "https://" + nodeIP + ":" + nodePort + argPath
}

func main() {
	argURL := parseArgs(os.Args[1:])
	fmt.Println("url =", argURL)
	
	rate := 100
	okRate := 1
	var nokRate int

	// first, find the point at which the system breaks
	for {
		if RunTest(argURL, rate) {
			okRate = rate
			rate *= 2
		} else {
			nokRate = rate
			break
		}
	}

	// next, do a binary search between okRate and nokRate
	for (nokRate - okRate) > 1 {
		rate = (nokRate + okRate) / 2
		if RunTest(argURL, rate) {
			okRate = rate
		} else {
			nokRate = rate
		}
	}
	fmt.Printf("➡️Maximum Working Rate: %d req/sec\n", okRate)
}
