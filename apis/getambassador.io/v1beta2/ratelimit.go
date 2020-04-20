package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RateLimit struct {
	*metaV1.TypeMeta
	*metaV1.ObjectMeta `json:"metadata"`
	Spec               *RateLimitSpec `json:"spec"`
}

type RateLimitSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`
	Domain       string       `json:"domain"`
	Limits       []Limit      `json:"limits"`
}

type Limit struct {
	Pattern []map[string]string `json:"pattern"`
	Rate    uint64              `json:"rate"`
	Unit    string              `json:"unit"`
	Source  string              `json:"-"`
}
