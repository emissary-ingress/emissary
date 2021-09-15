package watt

import (
	"encoding/json"
	"time"

	"github.com/datawire/ambassador/v2/pkg/consulwatch"

	"github.com/datawire/ambassador/v2/pkg/k8s"
)

type ConsulSnapshot struct {
	Endpoints map[string]consulwatch.Endpoints `json:",omitempty"`
}

func (s *ConsulSnapshot) DeepCopy() (*ConsulSnapshot, error) {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	res := &ConsulSnapshot{}
	err = json.Unmarshal(jsonBytes, res)

	return res, err
}

type Error struct {
	Source    string
	Message   string
	Timestamp int64
}

func NewError(source, message string) Error {
	return Error{Source: source, Message: message, Timestamp: time.Now().Unix()}
}

type Snapshot struct {
	Consul     ConsulSnapshot            `json:",omitempty"`
	Kubernetes map[string][]k8s.Resource `json:",omitempty"`
	Errors     map[string][]Error        `json:",omitempty"`
}
