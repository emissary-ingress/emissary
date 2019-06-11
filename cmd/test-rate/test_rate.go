package main

import (
	"flag"
	"fmt"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

func main() {

	rate := flag.Int("rate", 100, "RPS rate")
	url := flag.String("url", "", "URL to loadtest")
	flag.Parse()
	duration := 5 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    *url,
	})
	vegetaRate := vegeta.Rate{Freq: *rate, Per: time.Second}
	name := "atk-" + string(*rate)
	attacker := vegeta.NewAttacker()
	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, vegetaRate, duration, name) {
		metrics.Add(res)
	}
	metrics.Close()
	fmt.Printf("%f", metrics.Success)
	/*
		if metrics.Success < float64(1) {
			fmt.Printf("💥  Failed at %d req/sec (latency %s) (success rate: %f)\n", rate, latency, metrics.Success)
			fmt.Printf("Errors: %v\n", metrics.Errors)
			return false
		}
		fmt.Printf("✨  Success at %d req/sec (latency %s) (success rate: %f)\n", rate, latency, metrics.Success)
		return true
	*/
}
