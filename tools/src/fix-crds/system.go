package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
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
	var args Args

	flagset := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	var help bool
	flagset.StringVar(&args.Target, "target", "", fmt.Sprintf("What will be consuming the YAML; one of %q", Targets))
	flagset.BoolVarP(&help, "help", "h", false, "Display this help text")
	err := flagset.Parse(strs)
	if help {
		fmt.Printf("Usage: %s [FLAGS] --target=TARGET [INPUT_FILES...]\n", os.Args[0])
		fmt.Println()
		fmt.Println("If no INPUT_FILES are given, input is read from stdin.")
		fmt.Println()
		fmt.Println("FLAGS:")
		fmt.Println(flagset.FlagUsagesWrapped(70))
		os.Exit(0)
	}
	if err != nil {
		return Args{}, err
	}

	if !inArray(args.Target, Targets) {
		return Args{}, fmt.Errorf("invalid --target=%q, valid values are %q", args.Target, Targets)
	}

	for _, path := range flagset.Args() {
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

	var crdNames []string
	for i := range crds {
		if err := FixCRD(args, &(crds[i])); err != nil {
			return err
		}
		crdNames = append(crdNames, crds[i].Metadata.Name)
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
	if err := writeAPIExt(output, args, crdNames); err != nil {
		return err
	}

	return nil
}

// VisitSchemaFunc is a callback to be passed to VisitAllSchemaProps.
//
// As a special case, if the returned error is ErrExcludeFromSchema, then the node is excluded from
// the schema and the appropriate `.XPreserveUnknownFields` is set.
type VisitSchemaFunc func(version string, node *apiext.JSONSchemaProps) error

var ErrExcludeFromSchema = errors.New("exclude from schema")

func VisitAllSchemaProps(crd *CRD, callback VisitSchemaFunc) error {
	if crd == nil {
		return nil
	}
	if crd.Spec.Validation != nil {
		err := visitAllSchemaProps("validation", crd.Spec.Validation.OpenAPIV3Schema, callback)
		if errors.Is(err, ErrExcludeFromSchema) {
			crd.Spec.Validation = nil
			err = nil
		}
		if err != nil {
			return fmt.Errorf(".spec.validation: %w", err)
		}
	}
	for _, version := range crd.Spec.Versions {
		if version.Schema != nil {
			err := visitAllSchemaProps(version.Name, version.Schema.OpenAPIV3Schema, callback)
			if errors.Is(err, ErrExcludeFromSchema) {
				version.Schema.OpenAPIV3Schema = nil
				err = nil
			}
			if err != nil {
				return fmt.Errorf(".spec.version.find(x=>(x.name==%q)).schema.openAPIV3Schema: %w", version.Name, err)
			}
		}
	}
	return nil
}

func visitAllSchemaProps(version string, root *apiext.JSONSchemaProps, callback VisitSchemaFunc) error {
	if root == nil {
		return nil
	}
	if err := callback(version, root); err != nil {
		return err
	}
	if root.Items != nil {
		if err := visitAllSchemaProps(version, root.Items.Schema, callback); err != nil {
			return fmt.Errorf(".items: %w", err)
		}
		for i := range root.Items.JSONSchemas {
			if err := visitAllSchemaProps(version, &(root.Items.JSONSchemas[i]), callback); err != nil {
				return fmt.Errorf(".items[%d]: %w", i, err)
			}
		}
	}
	for i := range root.AllOf {
		if err := visitAllSchemaProps(version, &(root.AllOf[i]), callback); err != nil {
			return fmt.Errorf(".allOf[%d]: %w", i, err)
		}
	}
	for i := range root.OneOf {
		if err := visitAllSchemaProps(version, &(root.OneOf[i]), callback); err != nil {
			return fmt.Errorf(".oneOf[%d]: %w", i, err)
		}
	}
	for i := range root.AnyOf {
		if err := visitAllSchemaProps(version, &(root.AnyOf[i]), callback); err != nil {
			return fmt.Errorf(".anyOf[%d]: %w", i, err)
		}
	}
	if err := visitAllSchemaProps(version, root.Not, callback); err != nil {
		return fmt.Errorf(".not: %w", err)
	}
	for k, v := range root.Properties {
		if err := visitAllSchemaProps(version, &v, callback); errors.Is(err, ErrExcludeFromSchema) {
			delete(root.Properties, k)
			val := true
			root.XPreserveUnknownFields = &val
		} else if err != nil {
			return fmt.Errorf(".properties[%q]: %w", k, err)
		} else {
			root.Properties[k] = v
		}
	}
	if root.AdditionalProperties != nil {
		if err := visitAllSchemaProps(version, root.AdditionalProperties.Schema, callback); errors.Is(err, ErrExcludeFromSchema) {
			root.AdditionalProperties = nil
			val := true
			root.XPreserveUnknownFields = &val
		} else if err != nil {
			return fmt.Errorf(".additionalProperties: %w", err)
		}
	}
	for k, v := range root.PatternProperties {
		if err := visitAllSchemaProps(version, &v, callback); errors.Is(err, ErrExcludeFromSchema) {
			delete(root.PatternProperties, k)
			val := true
			root.XPreserveUnknownFields = &val
		} else if err != nil {
			return fmt.Errorf(".patternProperties[%q]: %w", k, err)
		} else {
			root.PatternProperties[k] = v
		}
	}
	for k := range root.Dependencies {
		if err := visitAllSchemaProps(version, root.Dependencies[k].Schema, callback); err != nil {
			return fmt.Errorf(".depenencies[%q]: %w", k, err)
		}
	}
	if root.AdditionalItems != nil {
		if err := visitAllSchemaProps(version, root.AdditionalItems.Schema, callback); err != nil {
			return fmt.Errorf(".additionalItems: %w", err)
		}
	}
	for k, v := range root.Definitions {
		if err := visitAllSchemaProps(version, &v, callback); err != nil {
			return fmt.Errorf(".definitions[%q]: %w", k, err)
		}
		root.Definitions[k] = v
	}
	return nil
}
