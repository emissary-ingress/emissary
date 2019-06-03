package oidc_core_client

import (
	"encoding/json"

	jwt "github.com/dgrijalva/jwt-go"
)

// §2
type IDTokenClaims struct {
	Issuer                              *net.URL
	Subject                             string
	Audiences                           []string
	ExpiresAt                           time.Time
	IssuedAt                            time.Time
	AuthTime                            time.Time
	Nonce                               string
	AuthenticationContextClassReference string
	AuthenticationMethodsReferences     []string
	AuthorizedParty                     string
}

type rawIDTokenClaims struct {
	Issuer                              jsonURL                `json:"iss"`
	Subject                             string                 `json:"sub"`
	Audiences                           jsonStringOrStringList `json:"aud"`
	ExpiresAt                           jsonUnixTime           `json:"exp"`
	IssuedAt                            jsonUnixTime           `json:"iat"`
	AuthTime                            jsonUnixTime           `json:"auth_time"`
	Nonce                               string                 `json:"nonce"`
	AuthenticationContextClassReference string                 `json:"arc"`
	AuthenticationMethodsReferences     []string               `json:"amr"`
	AuthorizedParty                     string                 `json:"azp"`
}

func (idt *IDTokenClaims) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*idt = nil
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
		AuthTime:                            raw.AuthTime.Value,
		Nonce:                               raw.Nonce,
		AuthenticationContextClassReference: raw.AuthenticationContextClassReference,
		AuthenticationMethodsReferences:     raw.AuthenticationMethodsReferences,
		AuthorizedParty:                     raw.AuthorizedParty,
	}
	return nil
}

// Valid implements jwt.Claims.
func (idt IDTokenClaims) Valid() error {
	// TODO
}

var _ jwt.Claims = IDTokenClaims{}
