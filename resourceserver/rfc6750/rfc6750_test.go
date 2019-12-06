package rfc6750_test

import (
	"net/http"
	"testing"

	"github.com/datawire/apro/resourceserver/rfc6750"
)

func TestGetFromHeader(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		Header        http.Header
		ExpectedToken string
		ExpectedError bool
	}{
		"none":                {Header: http.Header{ /*------------------------------------*/ }, ExpectedToken: "" /*----*/, ExpectedError: false},
		"emptyheader":         {Header: http.Header{"Authorization": {""} /*----------------*/}, ExpectedToken: "" /*----*/, ExpectedError: false},
		"emptytoken":          {Header: http.Header{"Authorization": {"Bearer "} /*---------*/}, ExpectedToken: "" /*----*/, ExpectedError: true},
		"plain":               {Header: http.Header{"Authorization": {"Bearer sometoken"} /**/}, ExpectedToken: "sometoken", ExpectedError: false},
		"nobearer":            {Header: http.Header{"Authorization": {"sometoken"} /*-------*/}, ExpectedToken: "" /*----*/, ExpectedError: true},
		"bearerlower":         {Header: http.Header{"Authorization": {"bearer sometoken"} /**/}, ExpectedToken: "sometoken", ExpectedError: false},
		"manyspaces":          {Header: http.Header{"Authorization": {"Bearer      sometoken"}}, ExpectedToken: "sometoken", ExpectedError: false},
		"params":              {Header: http.Header{"Authorization": {"Bearer foo=bar"} /*--*/}, ExpectedToken: "" /*----*/, ExpectedError: true},
		"illegalcharacter_sp": {Header: http.Header{"Authorization": {"Bearer foo bar"} /*--*/}, ExpectedToken: "" /*----*/, ExpectedError: true},
		"illegalcharacter_qm": {Header: http.Header{"Authorization": {"Bearer foo?bar"} /*--*/}, ExpectedToken: "" /*----*/, ExpectedError: true},
	}

	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			token, err := rfc6750.GetFromHeader(testData.Header)
			if token != testData.ExpectedToken {
				t.Errorf("token did not match expected value\n"+
					"\texpected: %q\n"+
					"\treceived: %q",
					testData.ExpectedToken,
					token)
			}
			if (err != nil) != testData.ExpectedError {
				if err == nil {
					t.Error("expected an error, but did not get one")
				} else {
					t.Errorf("did not expect an error, but got one: %v", err)
				}
			}
		})
	}
}
