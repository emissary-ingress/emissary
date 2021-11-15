package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Product string

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

type Args struct {
	Product     Product
	KubeVersion *semver.Version
	InputFiles  []*os.File
}

func ParseArgs(strs ...string) (Args, error) {
	if len(strs) < 2 {
		return Args{}, errors.Errorf("requires at least 2 arguments, got %d", len(strs))
	}

	args := Args{}

	for _, straw := range Products {
		if strs[0] == string(straw) {
			args.Product = straw
		}
	}
	if args.Product == "" {
		return Args{}, errors.Errorf("invalid product: %q not in %q", strs[0], Products)
	}

	var err error
	args.KubeVersion, err = semver.NewVersion(strs[1])
	if err != nil {
		return Args{}, errors.Wrap(err, "invalid kubeversion")
	}

	for _, path := range strs[2:] {
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
				return errors.Wrapf(err, "reading file %q", file.Name())
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
				return errors.Wrapf(err, "parsing file %q", file.Name())
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

func (args Args) HaveKubeversion(requiredVersion string) bool {
	return args.KubeVersion.Compare(semver.MustParse(requiredVersion)) >= 0
}

func VisitAllSchemaProps(crd *CRD, callback func(string, string, *apiext.JSONSchemaProps, *apiext.JSONSchemaProps) bool) {
	if crd == nil {
		return
	}
	if crd.Spec.Validation != nil {
		visitAllSchemaProps(crd.Spec.Names.Kind, "validation", crd.Spec.Validation.OpenAPIV3Schema, nil, callback)
	}
	for _, version := range crd.Spec.Versions {
		if version.Schema != nil {
			visitAllSchemaProps(crd.Spec.Names.Kind, version.Name, version.Schema.OpenAPIV3Schema, nil, callback)
		}
	}
}

func visitAllSchemaProps(
	crdName string,
	version string,
	root *apiext.JSONSchemaProps,
	parent *apiext.JSONSchemaProps,
	callback func(string, string, *apiext.JSONSchemaProps, *apiext.JSONSchemaProps) bool,
) bool {
	if root == nil {
		return true
	}

	if !callback(crdName, version, root, parent) {
		return false
	}

	if root.Items != nil {
		if !visitAllSchemaProps(crdName, version, root.Items.Schema, root, callback) {
			root.Items.Schema = nil
		}

		tmpItems := []apiext.JSONSchemaProps{}
		for i := range root.Items.JSONSchemas {
			if visitAllSchemaProps(crdName, version, &(root.Items.JSONSchemas[i]), root, callback) {
				tmpItems = append(tmpItems, root.Items.JSONSchemas[i])
			}
		}
		root.Items.JSONSchemas = tmpItems
	}

	tmpItems := []apiext.JSONSchemaProps{}
	for i := range root.AllOf {
		if visitAllSchemaProps(crdName, version, &(root.AllOf[i]), root, callback) {
			tmpItems = append(tmpItems, root.AllOf[i])
		}
	}
	root.AllOf = tmpItems

	tmpItems = []apiext.JSONSchemaProps{}
	for i := range root.OneOf {
		if visitAllSchemaProps(crdName, version, &(root.OneOf[i]), root, callback) {
			tmpItems = append(tmpItems, root.OneOf[i])
		}
	}
	root.OneOf = tmpItems

	tmpItems = []apiext.JSONSchemaProps{}
	for i := range root.AnyOf {
		if visitAllSchemaProps(crdName, version, &(root.AnyOf[i]), root, callback) {
			tmpItems = append(tmpItems, root.AnyOf[i])
		}
	}
	root.AnyOf = tmpItems

	if !visitAllSchemaProps(crdName, version, root.Not, root, callback) {
		root.Not = nil
	}

	for k, v := range root.Properties {
		if !visitAllSchemaProps(crdName, version, &v, root, callback) {
			delete(root.Properties, k)
		} else {
			root.Properties[k] = v
		}
	}

	if root.AdditionalProperties != nil {
		if !visitAllSchemaProps(crdName, version, root.AdditionalProperties.Schema, root, callback) {
			root.AdditionalProperties = nil
		}
	}

	for k, v := range root.PatternProperties {
		if !visitAllSchemaProps(crdName, version, &v, root, callback) {
			delete(root.PatternProperties, k)
		} else {
			root.PatternProperties[k] = v
		}
	}

	for k := range root.Dependencies {
		if !visitAllSchemaProps(crdName, version, root.Dependencies[k].Schema, root, callback) {
			delete(root.Dependencies, k)
		}
	}

	if root.AdditionalItems != nil {
		if !visitAllSchemaProps(crdName, version, root.AdditionalItems.Schema, root, callback) {
			root.AdditionalItems = nil
		}
	}

	for k, v := range root.Definitions {
		if !visitAllSchemaProps(crdName, version, &v, root, callback) {
			delete(root.Definitions, k)
		} else {
			root.Definitions[k] = v
		}
	}

	return true
}
