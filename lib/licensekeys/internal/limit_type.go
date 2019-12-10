//nolint:dupl // We cannot unify feature and limit because of UnmarshalJSON needs access to a separate global variable
package internal

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// This exists in a separate internal package from the rest of license
// keys, for maximum type safety: Nothing can accidentally create an
// uncorecognized LimitType string.

type LimitType struct {
	value string
}

func (lt LimitType) String() string {
	return lt.value
}

var limit_types = map[string]LimitType{}

func AddLimitType(name string) LimitType {
	if _, exists := limit_types[name]; exists {
		panic(errors.Errorf("limit type %q already registered", name))
	}
	limit_type := LimitType{value: name}
	limit_types[name] = limit_type
	return limit_type
}

var LimitTypeUnrecognized = AddLimitType("")

func (lt *LimitType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	for _, limit_type := range limit_types {
		if limit_type.value == str {
			*lt = limit_type
			return nil
		}
	}

	*lt = LimitTypeUnrecognized
	return nil
}

func (lt LimitType) MarshalJSON() ([]byte, error) {
	return json.Marshal(lt.value)
}

func ParselimitType(str string) (LimitType, bool) {
	limit_type, ok := limit_types[str]
	if !ok {
		return LimitTypeUnrecognized, false
	}
	return limit_type, true
}

func ListKnownLimitTypes() []string {
	ret := make([]string, 0, len(limit_types)-1)
	for _, lt := range limit_types {
		if lt == LimitTypeUnrecognized {
			continue
		}
		ret = append(ret, lt.value)
	}
	return ret
}
