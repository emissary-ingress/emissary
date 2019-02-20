package v1

import (
	"encoding/json"
)

type AmbassadorID []string

func (aid *AmbassadorID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*aid = nil
		return nil
	}

	var err error

	var list []string
	var single string

	if err = json.Unmarshal(data, &single); err == nil {
		*aid = AmbassadorID([]string{single})
	}

	if err = json.Unmarshal(data, &list); err == nil {
		*aid = AmbassadorID(list)
	}

	return err
}

func (aid AmbassadorID) Matches(envVar string) bool {
	if aid == nil {
		aid = []string{"default"}
	}
	for _, item := range aid {
		if item == envVar {
			return true
		}
	}
	return false
}
