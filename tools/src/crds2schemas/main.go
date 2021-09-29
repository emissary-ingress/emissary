package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/datawire/dlib/dlog"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s INPUT_FILE.yaml OUTPUT_DIR/\n", os.Args[0])
		os.Exit(2)
	}
	if err := Main(context.Background(), os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}

func readInput(inputFilename string) ([]apiext.CustomResourceDefinition, error) {
	inputFile, err := os.Open(inputFilename)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	yamlReader := utilyaml.NewYAMLReader(bufio.NewReader(inputFile))
	var crds []apiext.CustomResourceDefinition
	for {
		yamlBytes, err := yamlReader.Read()
		if err != nil {
			if err == io.EOF {
				return crds, nil
			}
			return nil, fmt.Errorf("read yaml: %w", err)
		}

		var crd *apiext.CustomResourceDefinition
		if err := yaml.UnmarshalStrict(yamlBytes, &crd); err != nil {
			return nil, fmt.Errorf("decode yaml: %w", err)
		}
		if crd == nil {
			continue
		}

		crds = append(crds, *crd)
	}
}

func toRawJSON(in interface{}) apiext.JSON {
	bs, _ := json.Marshal(in)
	return apiext.JSON{Raw: bs}
}

func Main(ctx context.Context, inputFilename, outputDir string) error {
	crds, err := readInput(inputFilename)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	if err := os.Mkdir(outputDir, 0777); err != nil {
		return err
	}

	apiVersion := filepath.Base(outputDir)

	for _, crd := range crds {
		outputFilename := filepath.Join(outputDir, crd.Spec.Names.Kind+".schema")
		dlog.Infof(ctx, "Generating %q...", outputFilename)

		var spec apiext.JSONSchemaProps
		if crd.Spec.Validation != nil {
			spec = crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"]
		} else {
			found := false
			for _, version := range crd.Spec.Versions {
				if version.Name == apiVersion {
					spec = version.Schema.OpenAPIV3Schema.Properties["spec"]
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("could not find a schema in %q", inputFilename)
			}
		}
		outputData := apiext.JSONSchemaProps{
			Schema: "http://json-schema.org/schema#",
			ID:     "https://getambassador.io/schemas/" + crd.Spec.Names.Singular + ".json",
			Type:   "object",

			Required: append([]string{"apiVersion", "kind", "name"}, spec.Required...),
			Properties: map[string]apiext.JSONSchemaProps{
				"apiVersion": {
					Enum: []apiext.JSON{toRawJSON(crd.Spec.Group + "/" + apiVersion)},
				},
				"kind": {
					Enum: []apiext.JSON{toRawJSON(crd.Spec.Names.Kind)},
				},
				"name": {
					Type: "string",
				},
				"namespace": {
					Type: "string",
				},
				"generation": {
					Type: "integer",
				},
				"metadata_labels": {
					Type: "object",
					AdditionalProperties: &apiext.JSONSchemaPropsOrBool{
						Schema: &apiext.JSONSchemaProps{
							Type: "string",
						},
					},
				},
			},
			AdditionalProperties: spec.AdditionalProperties,
			Definitions:          spec.Definitions,
		}
		for k, v := range spec.Properties {
			outputData.Properties[k] = v
		}

		outputBytes, err := json.MarshalIndent(outputData, "", "    ")
		if err != nil {
			return err
		}
		outputBytes = append(outputBytes, '\n')

		if err := ioutil.WriteFile(outputFilename, outputBytes, 0666); err != nil {
			return err
		}
	}

	return nil
}
