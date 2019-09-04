package internal

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// This exists in a separate internal package from the rest of license
// keys, for maximum type safety: Nothing can accidentally create an
// unrecognized Feature string.

type Feature struct {
	value string
}

func (f Feature) String() string {
	return f.value
}

var features = map[string]Feature{}

func AddFeature(name string) Feature {
	if _, ok := features[name]; ok {
		panic(errors.Errorf("feature %q already registered", name))
	}
	feature := Feature{value: name}
	features[name] = feature
	return feature
}

var FeatureUnrecognized = AddFeature("")

func (f *Feature) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	for _, feature := range features {
		if feature.value == str {
			*f = feature
			return nil
		}
	}
	*f = FeatureUnrecognized
	return nil
}

func (f Feature) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

func ParseFeature(str string) (Feature, bool) {
	feature, ok := features[str]
	if !ok {
		return FeatureUnrecognized, false
	}
	return feature, true
}

func ListKnownFeatures() []string {
	ret := make([]string, 0, len(features)-1)
	for _, f := range features {
		if f == FeatureUnrecognized {
			continue
		}
		ret = append(ret, f.value)
	}
	return ret
}
