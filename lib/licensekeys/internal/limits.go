//nolint:dupl // We cannot unify feature and limit because of UnmarshalJSON needs access to a separate global variable
package internal

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// This exists in a separate internal package from the rest of license
// keys, for maximum type safety: Nothing can accidentally create an
// unrecognized Limit string.

type Limit struct {
	value string
}

func (f Limit) String() string {
	return f.value
}

var limits = map[string]Limit{}

func AddLimit(name string) Limit {
	if _, ok := limits[name]; ok {
		panic(errors.Errorf("limit %q already registered", name))
	}
	limit := Limit{value: name}
	limits[name] = limit
	return limit
}

var LimitUnrecognized = AddLimit("")

func (f *Limit) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	for _, limit := range limits {
		if limit.value == str {
			*f = limit
			return nil
		}
	}
	*f = LimitUnrecognized
	return nil
}

func (f Limit) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

func ParseLimit(str string) (Limit, bool) {
	limit, ok := limits[str]
	if !ok {
		return LimitUnrecognized, false
	}
	return limit, true
}

func ListKnownLimits() []string {
	ret := make([]string, 0, len(limits)-1)
	for _, f := range limits {
		if f == LimitUnrecognized {
			continue
		}
		ret = append(ret, f.value)
	}
	return ret
}
