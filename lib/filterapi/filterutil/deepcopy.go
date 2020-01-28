package filterutil

import (
	"github.com/datawire/apro/lib/filterapi"
)

func DeepCopyRequest(in *filterapi.FilterRequest) (*filterapi.FilterRequest, error) {
	bs, err := in.Marshal()
	if err != nil {
		return nil, err
	}
	out := new(filterapi.FilterRequest)
	if err := out.Unmarshal(bs); err != nil {
		return nil, err
	}
	return out, nil
}

