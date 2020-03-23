package v1

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Filter struct {
	*metaV1.TypeMeta
	*metaV1.ObjectMeta `json:"metadata"`
	Spec               *FilterSpec   `json:"spec"`
	Status             *FilterStatus `json:"status"`

	UnwrappedSpec interface{} `json:"-"`
	Desc          string      `json:"-"`
}

type FilterSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`

	OAuth2   *FilterOAuth2   `json:",omitempty"`
	Plugin   *FilterPlugin   `json:",omitempty"`
	JWT      *FilterJWT      `json:",omitempty"`
	External *FilterExternal `json:",omitempty"`
}

const (
	FilterState_OK    = "OK"
	FilterState_Error = "Error"
)

type FilterStatus struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

func (filter *Filter) Validate(secretsGetter coreV1client.SecretsGetter, haveRedis bool) error {
	var err error
	filter.UnwrappedSpec, filter.Desc, err = filter.Spec.Validate(filter.GetName(), filter.GetNamespace(), secretsGetter, haveRedis)
	if err == nil {
		filter.Status = &FilterStatus{
			State: FilterState_OK,
		}
	} else {
		filter.Status = &FilterStatus{
			State:  FilterState_Error,
			Reason: err.Error(),
		}
	}
	return err
}

func kindCount(isKind map[string]bool) uint {
	var cnt uint
	for _, is := range isKind {
		if is {
			cnt++
		}
	}
	return cnt
}

func kindNames(isKind map[string]bool) []string {
	ret := make([]string, 0, len(isKind))
	for kind := range isKind {
		ret = append(ret, kind)
	}
	sort.Strings(ret)
	return ret
}

func (spec *FilterSpec) Validate(name, namespace string, secretsGetter coreV1client.SecretsGetter, haveRedis bool) (unwrappedSpec interface{}, desc string, err error) {
	var ret struct {
		Spec interface{}
		Desc string
		Err  error
	}
	defer func() {
		unwrappedSpec = ret.Spec
		desc = ret.Desc
		err = ret.Err
	}()
	if spec == nil {
		ret.Err = errors.New("spec must be set")
		return
	}

	isKind := map[string]bool{
		"OAuth2":   spec.OAuth2 != nil,
		"Plugin":   spec.Plugin != nil,
		"JWT":      spec.JWT != nil,
		"External": spec.External != nil,
	}
	if kindCount(isKind) != 1 {
		ret.Err = errors.Errorf("must specify exactly 1 of: %v", kindNames(isKind))
		return
	}

	switch {
	case spec.OAuth2 != nil:
		ret.Err = spec.OAuth2.Validate(namespace, secretsGetter)
		ret.Spec = *spec.OAuth2
		if ret.Err == nil && !haveRedis {
			ret.Err = errors.Errorf("filter disabled because Redis does not seem to be available")
		}
		if ret.Err == nil {
			switch spec.OAuth2.GrantType {
			case GrantType_AuthorizationCode:
				ret.Desc = fmt.Sprintf("oauth2_domain=%s, oauth2_client_id=%s", spec.OAuth2.Domain(), spec.OAuth2.ClientID)
			case GrantType_ClientCredentials:
				ret.Desc = fmt.Sprintf("oauth2_client_credentials=%s", spec.OAuth2.AuthorizationURL)
			default:
				panic("should not happen")
			}
		}
	case spec.Plugin != nil:
		ret.Err = spec.Plugin.Validate()
		ret.Spec = *spec.Plugin
		if ret.Err == nil {
			ret.Desc = fmt.Sprintf("plugin=%s", spec.Plugin.Name)
		}
	case spec.JWT != nil:
		ret.Err = spec.JWT.Validate(name + "." + namespace)
		ret.Spec = *spec.JWT
		if ret.Err == nil {
			ret.Desc = "jwt"
		}
	case spec.External != nil:
		ret.Err = spec.External.Validate()
		ret.Spec = *spec.External
		if ret.Err == nil {
			ret.Desc = fmt.Sprintf("external=%s", spec.External.AuthService)
		}
	default:
		panic("should not happen")
	}

	// nolint:nakedret
	return
}
