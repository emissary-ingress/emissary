package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	vegeta "github.com/tsenart/vegeta/lib"
)

var Args = struct {
	URL string

	MaxLatency     time.Duration
	MinSuccessRate float64
	EnableHTTP2    bool

	CooldownMaxLatency time.Duration
	CooldownRPS        uint
	CooldownRequests   uint64
}{}

func parseArgs(args []string) {
	parser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage := fmt.Sprintf(`Usage: %s [OPTIONS] URL
Attempt to determine the maximum load that URL can handle

You may specify and ordinary http:// or https:// URL to have direct
absolute control over what it speaks to.  Alternatively, you may
prefix the URL with "nodeport+" to have it resolve a NodePort service:

    nodeport+https://SERVICE[.NAMESPACE][:PORTNAME]/PATH

If no PORTNAME is specified with a nodeport+ url, "http" or "https" is
used (depending on the URL scheme).

TODO: support "loadbalancer+" for LoadBalancer services.
`, os.Args[0])

	generalParser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage += `
OPTIONS (general):

  "so I asked richard for a latency budget, and he suggested 40ms
  total as a starting point" -- rhs

`
	generalParser.DurationVar(&Args.MaxLatency, "max-latency", 40*time.Millisecond, "Maximum latency to consider successful during load-testing; use 0 to disable latency checks")
	generalParser.Float64Var(&Args.MinSuccessRate, "min-success-rate", 0.95, "The required success rate for a given phase")
	generalParser.BoolVar(&Args.EnableHTTP2, "enable-http2", true, "Whether to enable HTTP/2 if the remote supports it")
	help := false
	generalParser.BoolVarP(&help, "help", "h", false, "Show this message")
	usage += generalParser.FlagUsagesWrapped(70)
	parser.AddFlagSet(generalParser)

	cooldownParser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage += `
OPTIONS (cooldown):

  During cooldown periods, require ${cooldown-requests} consecutive
  successful requests with latency < ${cooldown-max-latency} at
  ${cooldown-rps} in order to be considered "cooled down".

  In order do avoid either stressing either Envoy or the local client
  with new TCP connections, the ${cooldown-rps} should be low enough
  that it is likely that ` + "`latency*rps < 1.0`" + `.  10ms latency
  seems reasonable under non-load, so 1.0/10ms gives us 100rps as the
  default.

  Anecdotally, 500 seems too short for reliable client-cooldown (but
  maybe the newer ${cooldown-max-latency} requirement means it's more
  reliable now?).  1000 seems too be spending too much time staring at
  the screen, waiting for it to move on.  800 is a decent-ish
  compromise?

`
	cooldownParser.DurationVar(&Args.CooldownMaxLatency, "cooldown-max-latency", 10*time.Millisecond, "Maximum latency to consider successful during cooldown; use 0 to disable latency checks")
	cooldownParser.UintVar(&Args.CooldownRPS, "cooldown-rps", 100, "Requests per second during cooldown")
	cooldownParser.Uint64Var(&Args.CooldownRequests, "cooldown-requests", 800, "When to consider cooldown complete")
	usage += cooldownParser.FlagUsagesWrapped(70)
	parser.AddFlagSet(cooldownParser)

	if err := parser.Parse(args); err != nil {
		errusage(err)
	}
	if help {
		io.WriteString(os.Stdout, usage)
		os.Exit(0)
	}
	if parser.NArg() != 1 {
		errusage(errors.Errorf("expected 1 argument, got %d: %q", parser.NArg(), parser.Args()))
	}
	uStr := parser.Arg(0)
	u, err := url.Parse(uStr)
	if err != nil {
		errusage(errors.Wrap(err, "bad URL"))
	}
	if strings.HasPrefix(u.Scheme, "nodeport+") {
		// parse out our bits of the URL
		var service, namespace string
		hostparts := strings.Split(u.Hostname(), ".")
		switch len(hostparts) {
		case 1:
			service = hostparts[0]
		case 2:
			service = hostparts[0]
			namespace = hostparts[1]
		default:
			errusage(errors.Errorf("invalid number of segments in %s://SERVICE[.NAMESPACE] URL hostname", u.Scheme))
		}
		scheme := strings.TrimPrefix(u.Scheme, "nodeport+")
		portname := u.Port()
		if portname == "" {
			portname = scheme
		}

		// use kubectl to resolve everything
		var nodeIP string
		cmd := exec.Command("kubectl", "config", "view", "--output=go-template", "--template={{range .clusters}}{{.cluster.server}}{{end}}")
		cmd.Stderr = os.Stderr
		bs, err := cmd.Output()
		if err != nil {
			errfatal(errors.Wrap(err, "kubectl config view"))
		}
		for _, line := range strings.Split(string(bs), "\n") {
			clusterURL, err := url.Parse(line)
			if err != nil {
				errfatal(errors.Wrap(err, "invalid server URL in kubeconfig"))
			}
			nodeIP = clusterURL.Hostname()
		}
		cmdargs := []string{"kubectl", "get", "service", service, "--output=go-template", fmt.Sprintf("--template={{range .spec.ports}}{{if eq .name %q}}{{.nodePort}}{{end}}{{end}}", portname)}
		if namespace != "" {
			cmdargs = append(cmdargs, "--namespace="+namespace)
		}
		cmd = exec.Command(cmdargs[0], cmdargs[1:]...)
		cmd.Stderr = os.Stderr
		bs, err = cmd.Output()
		if err != nil {
			errfatal(errors.Wrap(err, "kubectl get service"))
		}
		nodePort := strings.TrimSpace(string(bs))

		// build the new final URL
		u.Scheme = scheme
		u.Host = net.JoinHostPort(nodeIP, nodePort)
	}
	Args.URL = u.String()
}

func errusage(err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\nTry '%s --help' for more information.\n", os.Args[0], err, os.Args[0])
	os.Exit(2)
}

func errfatal(err error) {
	fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
	os.Exit(1)
}

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
	vegeta.TLSConfig(&tls.Config{InsecureSkipVerify: true}), // #nosec G402
	vegeta.HTTP2(true),
	vegeta.Connections(maxFiles()), // setting -1 or 0 for no-limit doesn't seemt to work?
)

var sourcePortRE = regexp.MustCompile(":[1-9][0-9]*->")

func openFiles() int {
	fis, _ := ioutil.ReadDir("/dev/fd/")
	return len(fis)
}

type TestCase struct {
	URL      string
	RPS      int
	Duration time.Duration
}

type TestResult struct {
	Rate        int
	Successes   uint64
	Limited     uint64
	Requests    uint64
	Latency     time.Duration
	Errors      map[string]uint64
	FilesBefore int
	FilesAfter  int
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
	var limited uint64
	errs := make(map[string]uint64)
	filesBefore := openFiles()
	for res := range attacker.Attack(targeter, vegetaRate, tc.Duration, name) {
		// vegeta.Metrics doesn't consider HTTP 429 ("Too Many Requests") to be a "success",
		// but for testing the rate limit service, we should.
		switch res.Code {
		case http.StatusOK:
			successes++
		case http.StatusTooManyRequests:
			successes++
			limited++
			res.Error = ""
		default:
			errstr := fmt.Sprintf("code=%03d error=%#v x-envoy-overloaded=%#v body=%#v",
				res.Code,
				sourcePortRE.ReplaceAllString(res.Error, ":XYZ->"),
				res.Header.Get("x-envoy-overloaded"),
				string(res.Body),
			)
			errs[errstr] = errs[errstr] + 1
		}
		metrics.Add(res)
	}
	metrics.Close()
	filesAfter := openFiles()

	return TestResult{tc.RPS, successes, limited, metrics.Requests, metrics.Latencies.P95, errs, filesBefore, filesAfter}
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
	mainline := fmt.Sprintf("%s at rate=%drps (latency %s) (success rate=%d/%d=%f) (rate-limited: %d) (open files: %d→%d)",
		prefix, r.Rate, r.Latency, r.Successes, r.Requests, r.SuccessRate(), r.Limited, r.FilesBefore, r.FilesAfter)
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
		result := RunTestRaw(TestCase{URL: url, RPS: rate, Duration: 5 * time.Second})
		runs++
		fmt.Println(result)
		// Let it cool down.  This is both (1) to let Ambassador cool down (after a
		// failure), but also (2) to let the client cool down and for any dangling but not
		// keep-alive connections die.
		var passed uint64
		for passed < Args.CooldownRequests {
			runtime.GC()
			cooldown := RunTestRaw(TestCase{URL: url, RPS: int(Args.CooldownRPS), Duration: time.Second})
			if cooldown.Passed() && cooldown.Latency < Args.CooldownMaxLatency {
				passed += cooldown.Requests
			} else {
				passed = 0
			}
			fmt.Printf("  cooldown: %d (latency %s) (open files: %d→%d)\n",
				passed, cooldown.Latency, cooldown.FilesBefore, cooldown.FilesAfter)
		}
		if result.Passed() {
			return true
		}
		// try it up to 3 times
		if runs == 3 {
			return false
		}
	}
}

func main() {
	parseArgs(os.Args[1:])
	fmt.Println("url =", Args.URL)

	rate := 100
	okRate := 1
	var nokRate int

	// first, find the point at which the system breaks
	for {
		if RunTest(Args.URL, rate) {
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
		if RunTest(Args.URL, rate) {
			okRate = rate
		} else {
			nokRate = rate
		}
	}
	fmt.Printf("➡️Maximum Working Rate: %d req/sec\n", okRate)
}
