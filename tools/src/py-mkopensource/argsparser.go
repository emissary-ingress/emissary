package main

import (
	"flag"
	"fmt"
)

type cliArgs struct {
	outputType OutputType
}

var outputType *string

type OutputType string

const (
	markdownOutputType OutputType = "markdown"
	jsonOutputType     OutputType = "json"
)

func parseArgs() (cliArgs, error) {
	args := cliArgs{}
	outputType = flag.String("output-type", string(markdownOutputType), fmt.Sprintf("Format used when "+
		"printing dependency information. One of: %s, %s", markdownOutputType, jsonOutputType))

	flag.Parse()

	if outputType == nil {
		return args, fmt.Errorf("--output-type must be one of '%s', '%s'", markdownOutputType, jsonOutputType)
	}

	args.outputType = OutputType(*outputType)

	if args.outputType != markdownOutputType && args.outputType != jsonOutputType {
		return args, fmt.Errorf("--output-type must be one of '%s', '%s'", markdownOutputType, jsonOutputType)
	}

	return args, nil
}
