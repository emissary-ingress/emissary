package main

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/datawire/apro/cmd/loadtest-generator/attack"
	"github.com/datawire/apro/cmd/loadtest-generator/metrics"
)

var Args = struct {
	URL         string
	EnableHTTP2 bool
	CSVFilename string

	LoadRPSResolution  uint
	LoadMinSuccessRate float64
	LoadMaxRPS         uint

	LoadUntilMinDuration          time.Duration
	LoadUntilMinSamples           uint
	LoadUntilMinLatencyConfidence float64
	LoadUntilMaxLatencyMargin     time.Duration

	CooldownRPS uint

	CooldownUntilMinSamples           uint
	CooldownUntilMinLatencyConfidence float64
	CooldownUntilMaxLatencyMargin     time.Duration
	CooldownUntilMaxLatency           time.Duration
}{}

func parseArgs(args []string) {
	parser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage := fmt.Sprintf(`Usage: %s [OPTIONS] URL
Attempt to determine the maximum load that URL can handle

It will start at 100rps and measure the latency.  It will then go up
in ${rps-resolution} steps until it starts seeing failures (such that
the success rate drops below ${min-success-rate}), or it reaches
${max-rps} (whichever comes first).

You may specify an ordinary http:// or https:// URL to have direct
absolute control over what it speaks to.  Alternatively, you may
prefix the URL with "nodeport+" to have it resolve a NodePort service:

    nodeport+https://SERVICE[.NAMESPACE][:PORTNAME]/PATH

If no PORTNAME is specified with a nodeport+ url, then "http" or
"https" is used (depending on the URL scheme).

TODO: support "loadbalancer+" for LoadBalancer services.
`, os.Args[0])

	generalParser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage += `
OPTIONS (general):

`
	help := false
	generalParser.BoolVarP(&help, "help", "h", false, "Show this message")
	generalParser.BoolVar(&Args.EnableHTTP2, "enable-http2", true, "Whether to enable HTTP/2 if the remote supports it")
	generalParser.StringVar(&Args.CSVFilename, "csv-file", "", "Write rps-vs-latency to this CSV file")
	usage += generalParser.FlagUsagesWrapped(70)
	parser.AddFlagSet(generalParser)

	loadParser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage += `
OPTIONS (load):

  During load periods it will make requests at a given RPS, until all
  3 conditions are met:

    1. It has made at least ${until-min-samples} requests.
    2. It has been running for at least ${until-min-duration}.
    3. It is ${until-min-latency-confidence} % sure that the mean
       latency is accurate to at least ± ${until-max-latency-margin}.

  Once these conditions are met, it will consider the service to have
  successfully handled that RPS, if the success rate is at least
  ${min-success-rate}.

`
	loadParser.UintVar(&Args.LoadRPSResolution, "load-rps-resolution", 50, "Granularity of RPS measurements")
	loadParser.Float64Var(&Args.LoadMinSuccessRate, "load-min-success-rate", 0.95, "The required success rate")
	loadParser.UintVar(&Args.LoadMaxRPS, "load-max-rps", 0, "Maximum RPS to test (0 for no maximum)")

	loadParser.DurationVar(&Args.LoadUntilMinDuration, "load-until-min-duration", 5*time.Second, "Run at a given RPS for at least this long")
	loadParser.UintVar(&Args.LoadUntilMinSamples, "load-until-min-samples", 1000, "Make at least this many requests at a given RPS")
	loadParser.Float64Var(&Args.LoadUntilMinLatencyConfidence, "load-until-min-latency-confidence", 0.99, "Be at least this sure of the latency margin")
	loadParser.DurationVar(&Args.LoadUntilMaxLatencyMargin, "load-until-max-latency-margin", 2*time.Millisecond, "Run until the mean latency is accurate to ± this")

	usage += loadParser.FlagUsagesWrapped(70)
	parser.AddFlagSet(loadParser)

	cooldownParser := pflag.NewFlagSet("", pflag.ContinueOnError)
	usage += `
OPTIONS (cooldown):

  Between loads, cooldown at ${rps} until all 3 conditions are met:

    1. It has made ${until-min-samples} consecutive successful
       requests.
    2. It is ${until-min-latency-confidence} % sure that the mean
       latency is accurate to at least ± ${until-max-latency-margin}.
    3. The p95 latency is at most ${until-max-latency}.

  In order do avoid either stressing either Envoy or the local client
  with new TCP connections, the ${rps} should be low enough that it is
  likely that ` + "`latency*rps < 1.0`" + `.  10ms latency seems
  reasonable under non-load, so 1.0/10ms gives us 100rps as the
  default.

`
	cooldownParser.UintVar(&Args.CooldownRPS, "cooldown-rps", 100, "Requests per second during cooldown")

	cooldownParser.UintVar(&Args.CooldownUntilMinSamples, "cooldown-until-min-samples", 500, "Required number of consecutive successful requests")
	cooldownParser.Float64Var(&Args.CooldownUntilMinLatencyConfidence, "lcooldown-until-min-latency-confidence", 0.99, "Be at least this sure of the latency margin")
	cooldownParser.DurationVar(&Args.CooldownUntilMaxLatencyMargin, "cooldown-until-max-latency-margin", 2*time.Millisecond, "Run until the mean latency is accurate to ± this")
	cooldownParser.DurationVar(&Args.CooldownUntilMaxLatency, "cooldown-until-max-latency", 10*time.Millisecond, "Run until the p95 latency at most this")

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

var csvFile *os.File

var prevP95Latency time.Duration

func RunLoad(rps uint) attack.TestResult {
	for {
		startTime := time.Now()
		result := attack.TestCase{
			URL: Args.URL,
			RPS: rps,

			ShouldStop: func(m metrics.MetricsReader, filesBefore int) bool {
				if m.CountRequests()%(5*rps) == 0 {
					fmt.Printf("? ------- at rate=%drps (latency p95=%s max=%s margin(%d%%)=±%s) (success rate=%d/%d=%f) (rate-limited: %d)\n",
						rps,
						m.LatencyQuantile(0.95), m.LatencyMax(), int(Args.LoadUntilMinLatencyConfidence*100), m.LatencyMargin(Args.LoadUntilMinLatencyConfidence),
						m.CountSuccesses(), m.CountRequests(), m.SuccessRate(),
						m.CountLimited())
					// If there's a really bad spike in latency that corresponds with a a spike in
					// file descriptors, go ahead and bail early, and let the general spike detector
					// prune it out below.  The confidence tests means that this run would
					// eventually stabalize, but it would take a pointlessly long time.
					//
					// Define a "really bad" spike as:
					//  - 10 file descriptors (same as below)
					//  - 2 s (1000× below)
					if m.LatencyQuantile(0.95) > prevP95Latency+(2*time.Second) && attack.OpenFiles() > filesBefore+10 {
						return true
					}
				}
				if m.CountRequests() < Args.LoadUntilMinSamples {
					return false
				}
				if time.Since(startTime) < Args.LoadUntilMinDuration {
					return false
				}
				if m.LatencyMargin(Args.LoadUntilMinLatencyConfidence) > Args.LoadUntilMaxLatencyMargin {
					return false
				}
				return true
			},
		}.Run()

		// Prune spikes in latency that correspond with spikes
		// in file descriptors.  That corresponds with TLS
		// handshakes, which we know perform poorly and we
		// don't want to include in these benchmarks.
		//
		// Define a spike as:
		//  - 10 file descriptors
		//  - 2 ms
		if result.FilesAfter > result.FilesBefore+10 && result.Metrics.LatencyQuantile(0.95) > prevP95Latency+(2*time.Millisecond) {
			continue
		}
		prevP95Latency = result.Metrics.LatencyQuantile(0.95)

		return result
	}
}

var prevLine = "BoGuS"
var inLine = false

func printdup(line string) {
	if line == prevLine {
		fmt.Printf(".")
		inLine = true
	} else {
		if inLine {
			fmt.Printf("\n")
		}
		fmt.Println(line)
		inLine = false
		prevLine = line
	}
}

func RunCooldown() {
	for {
		runtime.GC()
		result := attack.TestCase{
			URL: Args.URL,
			RPS: Args.CooldownRPS,

			ShouldStop: func(m metrics.MetricsReader, _ int) bool {
				if m.CountSuccesses() < m.CountRequests() {
					for errstr := range m.Errors() {
						printdup(fmt.Sprintf("  cooldown: error: %s", errstr))
					}
					return true
				}

				if m.CountRequests()%100 == 0 {
					printdup(fmt.Sprintf("  cooldown: requests=%d latency-p95=%v latency-margin(c=%d%%)=%v",
						m.CountRequests(),
						m.LatencyQuantile(0.95),
						int(Args.CooldownUntilMinLatencyConfidence*100),
						m.LatencyMargin(Args.CooldownUntilMinLatencyConfidence)))
				}

				if m.CountRequests() < Args.CooldownUntilMinSamples {
					return false
				}
				if m.LatencyMargin(Args.CooldownUntilMinLatencyConfidence) > Args.CooldownUntilMaxLatencyMargin {
					return false
				}
				if m.LatencyQuantile(0.95) > Args.CooldownUntilMaxLatency {
					return false
				}
				return true
			},
		}.Run()
		if result.Metrics.CountSuccesses() == result.Metrics.CountRequests() {
			break
		}
	}
	printdup("")
}

func Run(rate uint) bool {
	result := RunLoad(rate)
	fmt.Println(result.String(Args.LoadMinSuccessRate, Args.LoadUntilMinLatencyConfidence))
	RunCooldown()
	if result.Passed(Args.LoadMinSuccessRate) {
		if csvFile != nil {
			fmt.Fprintf(csvFile, "%d,%f,%f\n",
				result.Rate,
				result.Metrics.LatencyQuantile(0.95).Seconds()*1000,
				result.Metrics.LatencyMax().Seconds()*1000)
		}
		return true
	}
	return false
}

func main() {
	parseArgs(os.Args[1:])
	fmt.Println("url =", Args.URL)
	attack.SetHTTP2Enabled(Args.EnableHTTP2)
	if Args.CSVFilename != "" {
		var err error
		csvFile, err = os.OpenFile(Args.CSVFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			errfatal(err)
		}
		fmt.Fprintln(csvFile, "rate (req/sec),p95-latency (ms),max-latency (ms)")
	}

	rate := uint(100)
	okRate := uint(1)

	// first, find the point at which the system breaks
	for Args.LoadMaxRPS == 0 || rate <= Args.LoadMaxRPS {
		if Run(rate) {
			okRate = rate
			rate += Args.LoadRPSResolution
		} else {
			break
		}
	}

	fmt.Printf("➡️Maximum Working Rate: %d req/sec\n", okRate)
	if okRate == 1 {
		os.Exit(1)
	}
}
