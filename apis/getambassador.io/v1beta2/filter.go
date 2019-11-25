package v1

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	coreV1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type FilterSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`

	OAuth2   *FilterOAuth2   `json:",omitempty"`
	Plugin   *FilterPlugin   `json:",omitempty"`
	JWT      *FilterJWT      `json:",omitempty"`
	External *FilterExternal `json:",omitempty"`
	Internal *FilterInternal `json:",omitempty"`
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

type FilterInfo struct {
	Spec interface{}
	Desc string
	Err  error
}

func (spec *FilterSpec) Validate(namespace string, secretsGetter coreV1client.SecretsGetter, haveRedis bool) FilterInfo {
	var ret FilterInfo
	if spec == nil {
		ret.Err = errors.New("spec must be set")
		return ret
	}

	isKind := map[string]bool{
		"OAuth2":   spec.OAuth2 != nil,
		"Plugin":   spec.Plugin != nil,
		"JWT":      spec.JWT != nil,
		"External": spec.External != nil,
		"Internal": spec.Internal != nil,
	}
	if kindCount(isKind) != 1 {
		ret.Err = errors.Errorf("must specify exactly 1 of: %v", kindNames(isKind))
		return ret
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
		ret.Err = spec.JWT.Validate()
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
	case spec.Internal != nil:
		ret.Spec = *spec.Internal
		ret.Desc = "internal"
	default:
		panic("should not happen")
	}

	return ret
}
