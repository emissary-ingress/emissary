package main

import (
	"flag"
	"fmt"
)

type OutputType string
type ApplicationType string

type cliArgs struct {
	outputType      *OutputType
	applicationType *ApplicationType
}

const (
	markdownOutputType OutputType = "markdown"
	jsonOutputType     OutputType = "json"

	// Validations to do on the licenses.
	// The only validation for "internal" is to check chat forbidden licenses are not used
	internalApplication ApplicationType = "internal"
	// "external" applications have additional license requirements as documented in
	//https://www.notion.so/datawire/License-Management-5194ca50c9684ff4b301143806c92157
	externalApplication ApplicationType = "external"
)

func parseArgs() (cliArgs, error) {
	var outputType *string
	var applicationType *string

	args := cliArgs{}
	outputType = flag.String("output-type", string(markdownOutputType), fmt.Sprintf("Format used when "+
		"printing dependency information. One of: %s, %s", markdownOutputType, jsonOutputType))
	applicationType = flag.String("application-type", string(externalApplication),
		fmt.Sprintf("Where will the application run. One of: %s, %s\n"+
			"Internal applications are run on Ambassador servers.\n"+
			"External applications run on customer machines", internalApplication, externalApplication))

	flag.Parse()

	var err error
	if args.outputType, err = getOutputType(outputType); err != nil {
		return args, err
	}

	if args.applicationType, err = getApplicationType(applicationType); err != nil {
		return args, err
	}

	return args, nil
}

func getOutputType(value *string) (*OutputType, error) {
	if value == nil {
		return nil, fmt.Errorf("--output-type must be one of '%s', '%s'", markdownOutputType, jsonOutputType)
	}

	outputType := OutputType(*value)
	if outputType != markdownOutputType && outputType != jsonOutputType {
		return nil, fmt.Errorf("--output-type must be one of '%s', '%s'", markdownOutputType, jsonOutputType)
	}

	return &outputType, nil
}

func getApplicationType(value *string) (*ApplicationType, error) {
	if value == nil {
		return nil, fmt.Errorf("--application-type must be one of '%s', '%s'", internalApplication, externalApplication)
	}

	applicationType := ApplicationType(*value)
	if applicationType != internalApplication && applicationType != externalApplication {
		return nil, fmt.Errorf("--application-type must be one of '%s', '%s'", internalApplication, externalApplication)
	}

	return &applicationType, nil
}
