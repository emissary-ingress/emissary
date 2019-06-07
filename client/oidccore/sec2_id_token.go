package oidccore

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// IDTokenClaims implements JWT-claim handling for an ID token, as specified by §2.
type IDTokenClaims struct {
	Issuer                              *url.URL  `json:"iss"`
	Subject                             string    `json:"sub"`
	Audiences                           []string  `json:"aud"`
	ExpiresAt                           time.Time `json:"exp"`
	IssuedAt                            time.Time `json:"iat"`
	AuthenticatedAt                     time.Time `json:"auth_time"`
	Nonce                               string    `json:"nonce"`
	AuthenticationContextClassReference string    `json:"arc"`
	AuthenticationMethodsReferences     []string  `json:"amr"`
	AuthorizedParty                     string    `json:"azp"`
}

type rawIDTokenClaims struct {
	Issuer                              jsonURL                `json:"iss"`
	Subject                             string                 `json:"sub"`
	Audiences                           jsonStringOrStringList `json:"aud"`
	ExpiresAt                           jsonUnixTime           `json:"exp"`
	IssuedAt                            jsonUnixTime           `json:"iat"`
	AuthenticatedAt                     jsonUnixTime           `json:"auth_time"`
	Nonce                               string                 `json:"nonce"`
	AuthenticationContextClassReference string                 `json:"arc"`
	AuthenticationMethodsReferences     []string               `json:"amr"`
	AuthorizedParty                     string                 `json:"azp"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (idt *IDTokenClaims) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*idt = IDTokenClaims{}
		return nil
	}

	var raw rawIDTokenClaims
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	*idt = IDTokenClaims{
		Issuer:                              raw.Issuer.Value,
		Subject:                             raw.Subject,
		Audiences:                           raw.Audiences.Value,
		ExpiresAt:                           raw.ExpiresAt.Value,
		IssuedAt:                            raw.IssuedAt.Value,
		AuthenticatedAt:                     raw.AuthenticatedAt.Value,
		Nonce:                               raw.Nonce,
		AuthenticationContextClassReference: raw.AuthenticationContextClassReference,
		AuthenticationMethodsReferences:     raw.AuthenticationMethodsReferences,
		AuthorizedParty:                     raw.AuthorizedParty,
	}
	return nil
}

//type idtokenContext struct {
//	ClientID         string
//	AuthTimeRequired bool
//	Nonce            string
//}

// Valid implements jwt.Claims.
func (idt IDTokenClaims) Valid() error {
	now := time.Now()

	if !strings.EqualFold(idt.Issuer.Scheme, "https") {
		return errors.Errorf("claim %q URL must use %q scheme: %q", "iss",
			"https", idt.Issuer.String())
	}
	if idt.Issuer.RawQuery != "" {
		return errors.Errorf("claim %q URL must not include a query component: %q",
			"iss", idt.Issuer.String())
	}
	if idt.Issuer.Fragment != "" {
		return errors.Errorf("claim %q URL must not include a fragment component: %q",
			"iss", idt.Issuer.String())
	}

	if len(idt.Subject) > 255 {
		return errors.Errorf("claim %q must not exceed 255 octets in length: %q",
			"sub", idt.Subject)
	}

	//if !inArray(ctx.ClientID.Audiences) {
	//	return errors.Errorf("claim %q does not contain client_id %q",
	//		"aud", ctx.ClientID)
	//}

	if idt.ExpiresAt.Before(now) {
		return errors.Errorf("claim %q is expired: exp=%v < now=%v",
			"exp", idt.ExpiresAt, now)
	}

	if !idt.IssuedAt.Before(now) {
		return errors.Errorf("claim %q was issued in the future: iat=%v > now=%v",
			"iat", idt.IssuedAt, now)
	}

	if idt.AuthenticatedAt.IsZero() {
		//if ctx.AuthTimeRequired {
		//	return errors.Errorf("claim %q was requested but is not present",
		//		"auth_time")
		//}
	} else {
		if !idt.AuthenticatedAt.Before(now) {
			return errors.Errorf("claim %q was issued in the future: auth_time=%v > now=%v",
				"auth_time", idt.AuthenticatedAt, now)
		}
	}

	//if ctx.Nonce != "" && idt.Nonce != ctx.Nonce {
	//	return errors.Errorf("claim %d doesn't match: requested=%q != received=%q",
	//		"nonce", ctx.Nonce, idt.Nonce)
	//}

	// The specification of azp is a clusterfuck
	// https://bitbucket.org/openid/connect/issues/973/
	//if idt.AuthorizedParty != "" && idt.AuthorizedParty != ctx.ClientID {
	//	return errors.Errorf("claim %q does not match client_id %q: %q",
	//		"azp", ctx.ClientID, idt.AuthorizedParty)
	//}

	return nil
}

var _ jwt.Claims = IDTokenClaims{}
