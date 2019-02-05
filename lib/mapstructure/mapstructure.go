// Package mapstructure converts arbitrary JSON-ish interface{}s to
// native Go structures.
//
// It is like github.com/mitchellh/mapstructure in concept (but not
// API), but follows encoding/json struct tag semantics.
package mapstructure

import (
	"encoding/json"
)

func Convert(in interface{}, out interface{}) error {
	jsonBytes, err := json.Marshal(in)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, out)
}
