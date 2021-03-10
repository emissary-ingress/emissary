package v1

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	envoyRatelimitV2 "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
)

type RateLimit struct {
	*metaV1.TypeMeta
	*metaV1.ObjectMeta `json:"metadata"`
	Spec               *RateLimitSpec `json:"spec"`
}

func (rl *RateLimit) Validate() error {
	qname := rl.GetName() + "." + rl.GetNamespace()
	if err := rl.Spec.Validate(qname); err != nil {
		return errors.Wrap(err, "spec")
	}
	return nil
}

type RateLimitSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`
	Domain       string       `json:"domain"`
	Limits       []Limit      `json:"limits"`
}

func (s *RateLimitSpec) Validate(qname string) error {
	if s == nil {
		return errors.New("not set")
	}
	if s.Domain == "" {
		return errors.Wrap(errors.New("cannot have empty domain"), "domain")
	}
	for i := range s.Limits {
		if err := s.Limits[i].Validate(); err != nil {
			return errors.Wrapf(err, "limits[%d]", i)
		}
		s.Limits[i].Source = qname
	}
	return nil
}

type Limit struct {
	// QName of the resource containing this limit; for use in
	// error messages.
	Source string `json:"-"`

	Name    string              `json:"name"`
	Pattern []map[string]string `json:"pattern"`
	Rate    uint32              `json:"rate"`
	Unit    RateLimitUnit       `json:"unit"`
	Action  RateLimitAction     `json:"action"`

	InjectResponseHeaders []HeaderFieldTemplate `json:"injectResponseHeaders"`
	InjectRequestHeaders  []HeaderFieldTemplate `json:"injectRequestHeaders"`
	ErrorResponse         ErrorResponse         `json:"errorResponse"`
}

func (l *Limit) Validate() error {
	if l == nil {
		return errors.New("not set")
	}
	for i, entry := range l.Pattern {
		if len(entry) == 0 {
			return errors.Wrapf(errors.New("empty entry"), "pattern[%d]", i)
		}
	}
	if err := l.Unit.Validate(); err != nil {
		return errors.Wrap(err, "unit")
	}
	for i := range l.InjectResponseHeaders {
		hf := &(l.InjectResponseHeaders[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrapf(err, "responseHeaders[%d]", i)
		}
	}
	for i := range l.InjectRequestHeaders {
		hf := &(l.InjectRequestHeaders[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrapf(err, "requestHeaders[%d]", i)
		}
	}
	// Do not set a default body template nor response headers for ratelimit
	// errror responses. In a future version of Ambassador, we can revisit this.
	if err := l.ErrorResponse.ValidateWithoutDefaults(); err != nil {
		return errors.Wrap(err, "errorResponse")
	}
	return nil
}

type RateLimitAction struct {
	int
}

var (
	RateLimitAction_ENFORCE  = RateLimitAction{0}
	RateLimitAction_LOG_ONLY = RateLimitAction{1}
)

var (
	RateLimitAction_String_ENFORCE  = "Enforce"
	RateLimitAction_String_LOG_ONLY = "LogOnly"
)

func ParseRateLimitAction(strval string) (*RateLimitAction, error) {
	if strval == "" || strings.EqualFold(strval, RateLimitAction_String_ENFORCE) {
		return &RateLimitAction_ENFORCE, nil
	}
	if strings.EqualFold(strval, RateLimitAction_String_LOG_ONLY) {
		return &RateLimitAction_LOG_ONLY, nil
	}
	return nil, errors.Errorf("unknown RateLimitAction \"%v\"", strval)
}

func (a *RateLimitAction) ToString() string {
	if a == nil {
		return RateLimitAction_String_ENFORCE
	}

	if *a == RateLimitAction_ENFORCE {
		return RateLimitAction_String_ENFORCE
	}
	if *a == RateLimitAction_LOG_ONLY {
		return RateLimitAction_String_LOG_ONLY
	}
	return RateLimitAction_String_ENFORCE
}

func (a *RateLimitAction) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*a = RateLimitAction_ENFORCE
		return nil
	}

	var strval string
	if err := json.Unmarshal(data, &strval); err != nil {
		return err
	}

	val, err := ParseRateLimitAction(strval)
	if err != nil {
		return err
	}

	*a = *val
	return nil
}

func (a RateLimitAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(a)
}

func (a RateLimitAction) Validate() error {
	return nil
}

type RateLimitUnit struct {
	envoyRatelimitV2.RateLimitResponse_RateLimit_Unit
}

var (
	RateLimitUnit_UNKNOWN = RateLimitUnit{envoyRatelimitV2.RateLimitResponse_RateLimit_UNKNOWN}
	RateLimitUnit_SECOND  = RateLimitUnit{envoyRatelimitV2.RateLimitResponse_RateLimit_SECOND}
	RateLimitUnit_MINUTE  = RateLimitUnit{envoyRatelimitV2.RateLimitResponse_RateLimit_MINUTE}
	RateLimitUnit_HOUR    = RateLimitUnit{envoyRatelimitV2.RateLimitResponse_RateLimit_HOUR}
	RateLimitUnit_DAY     = RateLimitUnit{envoyRatelimitV2.RateLimitResponse_RateLimit_DAY}
)

func (u RateLimitUnit) Duration() time.Duration {
	// Will return 0 otherwise
	return map[RateLimitUnit]time.Duration{
		RateLimitUnit_SECOND: time.Second,
		RateLimitUnit_MINUTE: time.Minute,
		RateLimitUnit_HOUR:   time.Hour,
		RateLimitUnit_DAY:    24 * time.Hour,
	}[u]
}

func (u *RateLimitUnit) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*u = RateLimitUnit_UNKNOWN
		return nil
	}

	var strval string
	if err := json.Unmarshal(data, &strval); err != nil {
		return err
	}

	val, err := ParseRateLimitUnit(strval)
	if err != nil {
		return err
	}

	*u = val
	return nil
}

func ParseRateLimitUnit(strval string) (RateLimitUnit, error) {
	// This mimics `vendor-ratelimit/src/config/config_impl.go:rateLimitDescriptor.loadDescriptors()`
	int32val, present := envoyRatelimitV2.RateLimitResponse_RateLimit_Unit_value[strings.ToUpper(strval)]
	if !present {
		return RateLimitUnit_UNKNOWN, errors.Errorf("not a valid ratelimit unit: %q", strval)
	}

	return RateLimitUnit{envoyRatelimitV2.RateLimitResponse_RateLimit_Unit(int32val)}, nil
}

func (u RateLimitUnit) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u RateLimitUnit) Validate() error {
	if u == RateLimitUnit_UNKNOWN {
		return errors.New("must not be \"UNKNOWN\"")
	}
	return nil
}
