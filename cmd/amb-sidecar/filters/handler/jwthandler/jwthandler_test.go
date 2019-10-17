package jwthandler_test

import (
	"context"
	"strings"
	"testing"

	envoyAuthV2 "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2"
	"github.com/dgrijalva/jwt-go"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/jwthandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/testutil"
)

type TestHeader struct {
	Template string
	Expect   string
}

func TestJWTInjectHeaders(t *testing.T) {
	assert := &testutil.Assert{T: t}

	// build the test-case /////////////////////////////////////////////////
	requestID := "test-request"
	tokenStruct := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"sub":  "1234567890",
		"name": "John Doe",
		"iat":  1516239022,
	})
	tokenStruct.Header["extra"] = "so much"
	tokenString, err := tokenStruct.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NotError(err)

	testHeaders := map[string]TestHeader{
		"X-Fixed-String":        {Template: `Fixed String`, Expect: "Fixed String"},
		"X-Override":            {Template: `after`, Expect: "after"},
		"X-Token-String":        {Template: `{{.token.Raw}}`, Expect: tokenString},
		"X-Token-H-Alg":         {Template: `{{.token.Header.alg}}`, Expect: "none"},
		"X-Token-H-Typ":         {Template: `{{.token.Header.typ}}`, Expect: "JWT"},
		"X-Token-H-Extra":       {Template: `{{.token.Header.extra}}`, Expect: "so much"},
		"X-Token-C-Sub":         {Template: `{{.token.Claims.sub}}`, Expect: "1234567890"},
		"X-Token-C-Name":        {Template: `{{.token.Claims.name}}`, Expect: "John Doe"},
		"X-Token-C-Iat":         {Template: `{{.token.Claims.iat}}`, Expect: "1.516239022e+09"}, // don't expect numbers to always be formatted the same as input
		"X-Token-C-Iat-Decimal": {Template: `{{printf "%.0f" .token.Claims.iat}}`, Expect: "1516239022"},
		"X-Token-S":             {Template: `{{.token.Signature}}`, Expect: tokenString[strings.LastIndexByte(tokenString, '.')+1:]},
		"X-Authorization":       {Template: `Authenticated {{.token.Header.typ}}; sub={{.token.Claims.sub}}; name={{printf "%q" .token.Claims.name}}`, Expect: `Authenticated JWT; sub=1234567890; name="John Doe"`},
	}
	spec := crd.FilterJWT{
		ValidAlgorithms: []string{"none"},
	}
	for thName, th := range testHeaders {
		spec.InjectRequestHeaders = append(spec.InjectRequestHeaders, crd.JWTHeaderField{
			Name:  thName,
			Value: th.Template,
		})
	}
	assert.NotError(spec.Validate())

	// run the filter //////////////////////////////////////////////////////

	filter := &jwthandler.JWTFilter{
		Spec: spec,
	}
	request := &filterapi.FilterRequest{ // envoyAuthV2.AttributeContext
		Request: &envoyAuthV2.AttributeContext_Request{
			Http: &envoyAuthV2.AttributeContext_HttpRequest{
				Id: requestID,
				Headers: map[string]string{
					"Authorization": "Bearer " + tokenString,
					"X-Override":    "before",
				},
			},
		},
	}
	ctx := middleware.WithRequestID(middleware.WithLogger(context.Background(), types.WrapTB(t)), requestID)
	response, err := filter.Filter(ctx, request)
	assert.NotError(err)

	// inspect the result //////////////////////////////////////////////////

	requestMod, requestModOK := response.(*filterapi.HTTPRequestModification)
	if !requestModOK {
		t.Fatalf("filter response had wrong type: %[1]T(%#[1]v)", request)
	}
	filterutil.ApplyRequestModification(request, requestMod)

	for thName := range testHeaders {
		thName := thName // capture loop variable
		th := testHeaders[thName]
		t.Run(thName, func(t *testing.T) {
			assert := &testutil.Assert{T: t}
			assert.StrEQ(th.Expect, request.Request.Http.Headers[thName])
		})
	}
}

func TestJWTErrorResponse(t *testing.T) {
	assert := &testutil.Assert{T: t}

	spec := crd.FilterJWT{
		ValidAlgorithms: []string{"none"},
		ErrorResponse:   crd.ErrorResponse{RawBodyTemplate: "Some {{.Template}}"},
	}

	assert.NotError(spec.Validate())
	assert.StrEQ("application/json", spec.ErrorResponse.ContentType)
	if spec.ErrorResponse.BodyTemplate == nil {
		assert.T.Fatalf("Expected BodyTemplate to be parsed")
	}
}
