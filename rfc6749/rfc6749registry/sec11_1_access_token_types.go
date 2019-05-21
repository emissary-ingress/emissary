package rfc6749registry

import (
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// AccessTokenType TODO §11.1.1.
type AccessTokenType struct {
	Name                              string
	AdditionalTokenEndpointParameters []string
	ChangeController                  string
	SpecificationDocuments            []string

	ClientNeedsBody                       bool
	ClientAuthorizationForResourceRequest func(token string, body io.Reader) (http.Header, error)

	ResourceServerNeedsBody             bool
	ResourceServerValidateAuthorization func(http.Header, io.Reader) (bool, error)
}

// AccessTokenTypeClientDriver TODO
type AccessTokenTypeClientDriver interface {
	NeedsBody() bool
	AuthorizationForResourceRequest(token string, body io.Reader) (http.Header, error)
}

type accessTokenTypeClientDriver AccessTokenType

func (driver accessTokenTypeClientDriver) NeedsBody() bool { return driver.ClientNeedsBody }
func (driver accessTokenTypeClientDriver) AuthorizationForResourceRequest(token string, body io.Reader) (http.Header, error) {
	return driver.ClientAuthorizationForResourceRequest(token, body)
}

// AccessTokenTypeResourceServerDriver TODO
type AccessTokenTypeResourceServerDriver interface {
	NeedsBody() bool
	ValidateAuthorization(http.Header, io.Reader) (bool, error)
}

type accessTokenTypeResourceServerDriver AccessTokenType

func (driver accessTokenTypeResourceServerDriver) NeedsBody() bool { return driver.ClientNeedsBody }
func (driver accessTokenTypeResourceServerDriver) ValidateAuthorization(header http.Header, body io.Reader) (bool, error) {
	return driver.ResourceServerValidateAuthorization(header, body)
}

var accessTokenTypeRegistry = make(map[string]AccessTokenType)

// Register TODO
func (tokenType AccessTokenType) Register() {
	typeName := strings.ToLower(tokenType.Name)
	if _, set := accessTokenTypeRegistry[typeName]; set {
		panic(errors.Errorf("token_type=%q already registered", typeName))
	}
	accessTokenTypeRegistry[typeName] = tokenType
}

// GetAccessTokenTypeClientDriver TODO
func GetAccessTokenTypeClientDriver(typeName string) AccessTokenTypeClientDriver {
	tokenType, ok := accessTokenTypeRegistry[strings.ToLower(typeName)]
	if !ok {
		return nil
	}
	return accessTokenTypeClientDriver(tokenType)
}

// GetAccessTokenTypeResourceServerDriver TODO
func GetAccessTokenTypeResourceServerDriver(typeName string) AccessTokenTypeResourceServerDriver {
	tokenType, ok := accessTokenTypeRegistry[strings.ToLower(typeName)]
	if !ok {
		return nil
	}
	return accessTokenTypeResourceServerDriver(tokenType)
}
