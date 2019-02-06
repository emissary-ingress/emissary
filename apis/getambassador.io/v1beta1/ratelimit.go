package v1

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
