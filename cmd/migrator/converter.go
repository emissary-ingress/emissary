package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	// yaml "github.com/goccy/go-yaml"
	"gopkg.in/yaml.v2"
	k8syaml "sigs.k8s.io/yaml"

	// v3yaml "github.com/emissary-ingress/emissary/v3/pkg/yaml"

	// v3crds "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"

	v4crds "github.com/emissary-ingress/emissary/v3/pkg/api/emissary-ingress.dev/v4alpha1"
	v3json "github.com/emissary-ingress/emissary/v3/pkg/json"
)

type v4ConversionFunc func(nativeResource interface{}) error

type conversionInfo struct {
	nativeType reflect.Type
	toV4       v4ConversionFunc
}

type conversionMap map[string]conversionInfo

func (cMap conversionMap) addConversion(kind string, v interface{}, toV4 v4ConversionFunc) {
	nativeType := reflect.TypeOf(v)
	// fmt.Printf("add %s: %s - %s\n", kind, nativeType.Kind(), nativeType.Name())

	cMap[kind] = conversionInfo{nativeType, toV4}
}

func (cMap conversionMap) lookup(kind string) (bool, reflect.Type, v4ConversionFunc) {
	info, ok := cMap[kind]

	if !ok {
		return false, nil, nil
	}

	return true, info.nativeType, info.toV4
}

func fixAPIVersion(nativeResource interface{}) error {
	// Swap the apiVersion...
	reflectedValue := reflect.ValueOf(nativeResource).Elem()
	typeMetaField := reflectedValue.FieldByName("TypeMeta")

	if !typeMetaField.IsValid() {
		return fmt.Errorf("no TypeMeta field in %#v", nativeResource)
	}

	apiVersionField := typeMetaField.FieldByName("APIVersion")

	if !apiVersionField.IsValid() {
		return fmt.Errorf("no APIVersion field in %#v", nativeResource)
	}

	apiVersionField.SetString("emissary-ingress.dev/v4alpha1")

	return nil
}

func convertV3toV4(originalResource interface{}, typeMap conversionMap) (interface{}, error) {
	var unstructuredResource map[string]interface{}

	switch originalResource := originalResource.(type) {
	case map[string]interface{}, map[interface{}]interface{}:
		unstructuredResource = convertMap(originalResource).(map[string]interface{})

	default:
		return nil, fmt.Errorf("unsupported type %s: %#v", reflect.TypeOf(originalResource).Name(), originalResource)
	}

	// fmt.Printf("---\n")
	// fmt.Printf("%#v\n", unstructuredResource)

	// fmt.Printf("\nTypeMap:\n")
	// for k, v := range typeMap {
	// 	fmt.Printf("  %s: %s\n", k, v.nativeType.Name())
	// }

	// Grab the apiVersion and kind.
	apiVersion, ok := unstructuredResource["apiVersion"].(string)

	if !ok {
		return nil, fmt.Errorf("apiVersion not found or not a string: %#v", unstructuredResource)
	}

	// Is this a getambassador.io/v3alpha1 resource?
	if apiVersion != "getambassador.io/v3alpha1" {
		// Nope.
		return nil, fmt.Errorf("unknown apiVersion %s: %#v", apiVersion, unstructuredResource)
	}

	kind, ok := unstructuredResource["kind"].(string)

	if !ok {
		return nil, fmt.Errorf("kind not found or not a string: %#v", unstructuredResource)
	}

	ok, nativeType, toV4 := typeMap.lookup(kind)

	if !ok {
		return nil, fmt.Errorf("unknown kind %s: %#v", kind, unstructuredResource)
	}

	// fmt.Printf("kind %s: native type %s\n", kind, nativeType.Name())

	nativeResource := reflect.New(nativeType).Interface()
	// fmt.Printf("kind %s: native resource type %s\n", kind, reflect.TypeOf(nativeResource).Name())

	// Convert our unstructured resource to JSON...
	rawJSON, err := json.MarshalIndent(unstructuredResource, "", "  ")

	if err != nil {
		return nil, fmt.Errorf("couldn't convert unstructured to JSON? %s", err)
	}

	// fmt.Printf("V3: %s\n", string(rawJSON))

	// ...then unmarshal that JSON into our native resource, using the "v3" tag.
	err = v3json.Unmarshal(rawJSON, nativeResource)
	// err := v4crds.UnstructuredToNative(unstructuredResource, nativeResource, "v3")

	if err != nil {
		return nil, fmt.Errorf("error converting object to native %#v: %s", unstructuredResource, err)
	}

	// Run the conversion (if any).
	if toV4 == nil {
		toV4 = fixAPIVersion
	}

	err = toV4(nativeResource)

	if err != nil {
		return nil, fmt.Errorf("error converting object to v4 %#v: %s", unstructuredResource, err)
	}

	return nativeResource, nil

	// // Dump as YAML.
	// yamlBytes, err := k8syaml.Marshal(nativeResource)

	// if err != nil {
	// 	return nil, fmt.Errorf("error marshaling to YAML %#v: %s", unstructuredResource, err)
	// }

	// // fmt.Prinf("V4:\n")
	// fmt.Printf("---\n%s", string(yamlBytes))
	// return nil
}

// convertMap converts a map[interface{}]interface{} to a map[string]interface{}.
// It's kind of infuriating the YAML decoder requires this.
func convertMap(m interface{}) interface{} {
	var retval interface{}

	switch m := m.(type) {
	case []interface{}:
		newArray := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(m).Elem()), 0, len(m)).Interface().([]interface{})
		for _, e := range m {
			newArray = append(newArray, convertMap(e))
		}

		retval = newArray

	case map[interface{}]interface{}:
		newMap := make(map[string]interface{}, len(m))

		for key, value := range m {
			newMap[key.(string)] = convertMap(value)
		}

		retval = newMap

	default:
		retval = m
	}

	return retval
}

func convertSingleResource(unstructuredResource interface{}, typeMap conversionMap) (interface{}, error) {
	switch value := unstructuredResource.(type) {
	case []interface{}:
		// This is a list of resources. Loop through them.
		result := make([]interface{}, 0, len(value))

		for i, item := range value {
			// fmt.Printf("item %d: %#v\n", i, item)
			v4obj, error := convertSingleResource(item, typeMap)

			if error != nil {
				return nil, fmt.Errorf("failed to convert array item %d (%#v): %v", i, item, error)
			}

			result = append(result, v4obj)
		}

		return result, nil

	default:
		return convertV3toV4(unstructuredResource, typeMap)
	}
}

func main() {
	typeMap := make(conversionMap)

	typeMap.addConversion("AuthService", v4crds.AuthService{}, nil)
	typeMap.addConversion("DevPortal", v4crds.DevPortal{}, nil)
	typeMap.addConversion("Host", v4crds.Host{}, nil)
	typeMap.addConversion("Listener", v4crds.Listener{}, nil)
	typeMap.addConversion("LogService", v4crds.LogService{}, nil)
	typeMap.addConversion("Mapping", v4crds.Mapping{}, nil)
	typeMap.addConversion("Module", v4crds.Module{}, nil)
	typeMap.addConversion("Features", v4crds.Features{}, nil)
	typeMap.addConversion("RateLimitService", v4crds.RateLimitService{}, nil)
	typeMap.addConversion("KubernetesServiceResolver", v4crds.KubernetesServiceResolver{}, nil)
	typeMap.addConversion("KubernetesEndpointResolver", v4crds.KubernetesEndpointResolver{}, nil)
	typeMap.addConversion("ConsulResolver", v4crds.ConsulResolver{}, nil)
	typeMap.addConversion("TCPMapping", v4crds.TCPMapping{}, nil)
	typeMap.addConversion("TLSContext", v4crds.TLSContext{}, nil)
	typeMap.addConversion("TracingService", v4crds.TracingService{}, nil)

	if len(os.Args) < 2 {
		fmt.Println("Usage: converter <path1> <path2> ... <pathN>")
		return
	}

	for _, path := range os.Args[1:] {
		reader, err := os.Open(path)

		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", path, err)
			continue
		}

		decoder := yaml.NewDecoder(reader)

		// Loop through all the YAML documents
		for {
			// Decode the next YAML document into unstructuredResource.
			var unstructuredResource interface{}
			err := decoder.Decode(&unstructuredResource)

			if err == io.EOF {
				// End of the YAML input
				break
			} else if err != nil {
				fmt.Printf("Error decoding YAML: %v\n", err)
				continue
			}

			v4obj, error := convertSingleResource(unstructuredResource, typeMap)

			if error != nil {
				fmt.Printf("error converting resource %#v: %v\n", v4obj, error)
			}

			// Dump as YAML.
			yamlBytes, err := k8syaml.Marshal(v4obj)

			if err != nil {
				fmt.Printf("error marshaling to YAML %#v: %s\n", v4obj, err)
			} else {
				fmt.Printf("%s", string(yamlBytes))
			}
		}
	}
}
