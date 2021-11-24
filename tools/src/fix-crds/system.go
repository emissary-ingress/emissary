package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

type Args struct {
	Target     string
	InputFiles []*os.File
}

func ParseArgs(strs ...string) (Args, error) {
	if len(strs) < 1 {
		return Args{}, fmt.Errorf("requires at least 1 argument, got %d", len(strs))
	}

	args := Args{}

	args.Target = strs[0]
	if !inArray(args.Target, Targets) {
		return Args{}, fmt.Errorf("invalid TARGET %q, valid values are %q", args.Target, Targets)
	}

	for _, path := range strs[1:] {
		file, err := os.Open(path)
		if err != nil {
			return Args{}, err
		}
		args.InputFiles = append(args.InputFiles, file)
	}
	if len(args.InputFiles) == 0 {
		args.InputFiles = append(args.InputFiles, os.Stdin)
	}

	return args, nil
}

type NilableString struct {
	Value *string
}

var ExplicitNil = &NilableString{nil}

func NewNilableString(s string) *NilableString {
	return &NilableString{&s}
}

func (s *NilableString) UnmarshalJSON(bs []byte) error {
	if string(bs) == "null" {
		*s = NilableString{nil}
		return nil
	}

	var str string
	if err := json.Unmarshal(bs, &str); err != nil {
		return err
	}
	*s = NilableString{&str}
	return nil
}

func (s NilableString) MarshalJSON() ([]byte, error) {
	if s.Value == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(*s.Value)
}

func Main(args Args, output io.Writer) error {
	var crds []CRD
	for _, file := range args.InputFiles {
		yr := utilyaml.NewYAMLReader(bufio.NewReader(file))
		for {
			yamlbytes, err := yr.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("reading file %q: %w", file.Name(), err)
			}

			empty := true
			for _, line := range bytes.Split(yamlbytes, []byte("\n")) {
				if len(bytes.TrimSpace(bytes.SplitN(line, []byte("#"), 2)[0])) > 0 {
					empty = false
					break
				}
			}
			if empty {
				continue
			}

			var crd CRD
			if err := yaml.Unmarshal(yamlbytes, &crd); err != nil {
				return fmt.Errorf("parsing file %q: %w", file.Name(), err)
			}
			crds = append(crds, crd)
		}
	}

	for i := range crds {
		if err := FixCRD(args, &(crds[i])); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(output, "# GENERATED FILE: edits made by hand will not be preserved.\n"); err != nil {
		return err
	}
	for _, crd := range crds {
		if _, err := io.WriteString(output, "---\n"); err != nil {
			return err
		}
		yamlbytes, err := yaml.Marshal(crd)
		if err != nil {
			return err
		}
		if _, err := output.Write(yamlbytes); err != nil {
			return err
		}
	}

	return nil
}

func VisitAllSchemaProps(crd *CRD, callback func(*apiext.JSONSchemaProps)) {
	if crd == nil {
		return
	}
	if crd.Spec.Validation != nil {
		visitAllSchemaProps(crd.Spec.Validation.OpenAPIV3Schema, callback)
	}
	for _, version := range crd.Spec.Versions {
		if version.Schema != nil {
			visitAllSchemaProps(version.Schema.OpenAPIV3Schema, callback)
		}
	}
}

func visitAllSchemaProps(root *apiext.JSONSchemaProps, callback func(*apiext.JSONSchemaProps)) {
	if root == nil {
		return
	}
	callback(root)
	if root.Items != nil {
		visitAllSchemaProps(root.Items.Schema, callback)
		for i := range root.Items.JSONSchemas {
			visitAllSchemaProps(&(root.Items.JSONSchemas[i]), callback)
		}
	}
	for i := range root.AllOf {
		visitAllSchemaProps(&(root.AllOf[i]), callback)
	}
	for i := range root.OneOf {
		visitAllSchemaProps(&(root.OneOf[i]), callback)
	}
	for i := range root.AnyOf {
		visitAllSchemaProps(&(root.AnyOf[i]), callback)
	}
	visitAllSchemaProps(root.Not, callback)
	for k, v := range root.Properties {
		visitAllSchemaProps(&v, callback)
		root.Properties[k] = v
	}
	if root.AdditionalProperties != nil {
		visitAllSchemaProps(root.AdditionalProperties.Schema, callback)
	}
	for k, v := range root.PatternProperties {
		visitAllSchemaProps(&v, callback)
		root.PatternProperties[k] = v
	}
	for k := range root.Dependencies {
		visitAllSchemaProps(root.Dependencies[k].Schema, callback)
	}
	if root.AdditionalItems != nil {
		visitAllSchemaProps(root.AdditionalItems.Schema, callback)
	}
	for k, v := range root.Definitions {
		visitAllSchemaProps(&v, callback)
		root.Definitions[k] = v
	}
}
