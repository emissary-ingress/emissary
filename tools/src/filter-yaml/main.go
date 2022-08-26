// This script is to help generate any flat yaml files from the emissary helm chart.
//
// This script takes two arguments:
//  1. A multi-doc yaml file generated from running:
//     `helm template emissary -f [VALUES_FILE.yaml] -n [NAMESPACE] ./build-output/chart-{VER}_{CHART_VER}/`
//  2. A yaml file listing the required kubernetes resources from the generated helm template to
//     output to stdout. See ../aes/require.yaml for an example
//
// This script will output to stdout the resources from 1) iff they are referenced in 2). It will
// preserve the ordering from 2), and will error if any resources named in 2) are missing in 1)
package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"sigs.k8s.io/yaml"

	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func getResourceKey(resource kates.Object) string {
	return resource.GetObjectKind().GroupVersionKind().Kind +
		"." + resource.GetName() +
		"." + resource.GetNamespace()
}

func Keys(m map[string]kates.Object) []string {
	ret := make([]string, 0, len(m))
	for key := range m {
		ret = append(ret, key)
	}
	sort.Strings(ret)
	return ret
}

type Requirement struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

func (req Requirement) Key() string {
	return req.Kind +
		"." + req.Name +
		"." + req.Namespace
}

type Requirements struct {
	Resources        []Requirement `json:"resources"`
	DisableResources []Requirement `json:"disableResources"`
}

func Main(helmFilename, reqsFilename string, outFile io.Writer) error {
	helmBytes, err := os.ReadFile(helmFilename)
	if err != nil {
		return err
	}
	helmObjectList, err := kates.ParseManifestsToUnstructured(string(helmBytes))
	if err != nil {
		return err
	}
	templatedHelm := make(map[string]kates.Object, len(helmObjectList))
	for _, yamlDoc := range helmObjectList {
		templatedHelm[getResourceKey(yamlDoc)] = yamlDoc
	}

	reqsBytes, err := os.ReadFile(reqsFilename)
	if err != nil {
		return err
	}
	var reqs Requirements
	if err := yaml.Unmarshal(reqsBytes, &reqs); err != nil {
		return err
	}

	fmt.Fprintln(outFile, "# GENERATED FILE: edits made by hand will not be preserved.")
	// Print out required resources in the order they appear in require_file.  Order actually
	// matters here, for example, we need the namespace show up before any namespaced resources.
	for _, req := range reqs.Resources {
		fmt.Fprintln(outFile, "---")
		obj, ok := templatedHelm[req.Key()]
		if !ok {
			return fmt.Errorf("Resource %q not found in generated yaml (known resources are: %q)", req.Key(), Keys(templatedHelm))
		}
		delete(templatedHelm, req.Key())
		objBytes, err := yaml.Marshal(obj)
		if err != nil {
			return err
		}
		if _, err := outFile.Write(objBytes); err != nil {
			return err
		}
	}
	for _, req := range reqs.DisableResources {
		if _, ok := templatedHelm[req.Key()]; !ok {
			return fmt.Errorf("Resource %q not found in generated yaml (known resources are: %q)", req.Key(), Keys(templatedHelm))
		}
		delete(templatedHelm, req.Key())
	}
	if len(templatedHelm) > 0 {
		unusedKeys := make([]string, 0, len(templatedHelm))
		for key := range templatedHelm {
			unusedKeys = append(unusedKeys, key)
		}
		sort.Strings(unusedKeys)
		return fmt.Errorf("input file %q didn't specify how to handle the following %d resources: %v",
			reqsFilename, len(templatedHelm), unusedKeys)
	}

	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s HELM_GENERATED_FILE REQUIREMENTS_FILE >FILTERED_FILE\n", os.Args[0])
		os.Exit(2)
	}
	if err := Main(os.Args[1], os.Args[2], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
