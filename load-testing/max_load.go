package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tsenart/vegeta/lib"
)

var attacker = vegeta.NewAttacker()
var nodeIP string
var nodePort string

func rawTestRate(rate int) (success float64, latency time.Duration) {
	duration := 10 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://" + nodeIP + ":" + nodePort + "/http-echo/",
	})
	vegetaRate := vegeta.Rate{Freq: rate, Per: time.Second}
	name := "atk-" + string(rate)
	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, vegetaRate, duration, name) {
		metrics.Add(res)
	}
	metrics.Close()

	return metrics.Success, metrics.Latencies.P95
}

func testRate(rate int) bool {
	success, latency := rawTestRate(rate)
	if success < 1 {
		fmt.Printf("✘ Failed at %d req/sec (latency %s) (success rate: %f)\n", rate, latency, success)
		// let it cool down
		passed := 0
		for passed < 2 {
			s, _ := rawTestRate(1)
			if s < 1 {
				passed = 0
			} else {
				passed++
			}
		}
		return false
	}
	fmt.Printf("✔ Success at %d req/sec (latency %s) (success rate: %f)\n", rate, latency, success)
	return true
}

func main() {
	bs, _ := exec.Command("kubectl", "config", "view", "--output=go-template", "--template={{range .clusters}}{{.cluster.server}}{{end}}").Output()
	for _, line := range strings.Split(string(bs), "\n") {
		parts := strings.Split(strings.TrimPrefix(line, "https://"), ":")
		nodeIP = parts[0]
	}
	bs, _ = exec.Command("kubectl", "get", "service", "ambassador", "--output=go-template", "--template={{range .spec.ports}}{{if eq .name \"http\"}}{{.nodePort}}{{end}}{{end}}").Output()
	nodePort = strings.TrimSpace(string(bs))
	fmt.Printf("ambassador = %s:%s\n", nodeIP, nodePort)

	rate := 100
	okRate := 1
	var nokRate int

	// first, find the point at which the system breaks
	for {
		if testRate(rate) {
			okRate = rate
			fmt.Printf("!!!  Success at %d req/sec\n", rate)
			rate *= 2
		} else {
			nokRate = rate
			fmt.Printf(":(  Failed at %d req/sec\n", rate)
			break
		}
	}

	// next, do a binary search between okRate and nokRate
	for (nokRate - okRate) > 1 {
		rate = (nokRate + okRate) / 2
		if testRate(rate) {
			okRate = rate
			fmt.Printf("!!!  Success at %d req/sec\n", rate)
		} else {
			nokRate = rate
			fmt.Printf(":(  Failed at %d req/sec\n", rate)
		}
	}
	fmt.Printf("➡️  Maximum Working Rate: %d req/sec\n", okRate)
}
