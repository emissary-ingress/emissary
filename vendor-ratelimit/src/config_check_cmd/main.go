package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lyft/gostats"
	"github.com/lyft/ratelimit/src/config"
)

func loadConfigs(allConfigs []config.RateLimitConfigToLoad) {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("error loading rate limit configs: %s\n", err.(error).Error())
			os.Exit(1)
		}
	}()

	dummyStats := stats.NewStore(stats.NewNullSink(), false)
	config.NewRateLimitConfigImpl(allConfigs, dummyStats)
}

func main() {
	configDirectory := flag.String(
		"config_dir", "", "path to directory containing rate limit configs")
	flag.Parse()
	fmt.Printf("checking rate limit configs...\n")
	fmt.Printf("loading config directory: %s\n", *configDirectory)

	files, err := ioutil.ReadDir(*configDirectory)
	if err != nil {
		fmt.Printf("error opening directory %s: %s\n", *configDirectory, err.Error())
		os.Exit(1)
	}

	allConfigs := []config.RateLimitConfigToLoad{}
	for _, file := range files {
		finalPath := filepath.Join(*configDirectory, file.Name())
		fmt.Printf("opening config file: %s\n", finalPath)
		bytes, err := ioutil.ReadFile(finalPath)
		if err != nil {
			fmt.Printf("error reading file %s: %s\n", finalPath, err.Error())
			os.Exit(1)
		}
		allConfigs = append(allConfigs, config.RateLimitConfigToLoad{finalPath, string(bytes)})
	}

	loadConfigs(allConfigs)
	fmt.Printf("all rate limit configs ok\n")
}
