package rfc7235_test

import (
	"fmt"
	"testing"

	. "github.com/datawire/apro/common/rfc7235"
)

func TestParseCredentials(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		Input               string
		ExpectedCredentials Credentials
		ExpectedError       bool
	}{
		"example-rfc7617-2":   {`Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==`, Credentials{AuthScheme: `Basic`, Body: CredentialsLegacy(`QWxhZGRpbjpvcGVuIHNlc2FtZQ==`)}, false},
		"example-rfc7617-2.1": {`Basic dGVzdDoxMjPCow==` /*-------*/, Credentials{AuthScheme: `Basic`, Body: CredentialsLegacy(`dGVzdDoxMjPCow==` /*-------*/)}, false},
		"example-rfc6750-2.1": {`Bearer mF_9.B5f-4.1JqM` /*-------*/, Credentials{AuthScheme: `Bearer`, Body: CredentialsLegacy(`mF_9.B5f-4.1JqM` /*-------*/)}, false},
		// these are copied from the rfc6750 tests; not all of them are valid rfc6750 credentials, but rfc7235 doesn't know that.
		"emptyheader":         {"" /*-----------------------------*/, Credentials{ /*----------------------------------------------------------------------*/ }, true},
		"emptytoken":          {"Bearer " /*----------------------*/, Credentials{AuthScheme: `Bearer`, Body: CredentialsParameters{} /*--------------------*/}, false},
		"plain":               {"Bearer sometoken" /*-------------*/, Credentials{AuthScheme: `Bearer`, Body: CredentialsLegacy("sometoken" /*-------------*/)}, false},
		"nobearer":            {"sometoken" /*--------------------*/, Credentials{ /*----------------------------------------------------------------------*/ }, true},
		"bearerlower":         {"bearer sometoken" /*-------------*/, Credentials{AuthScheme: `bearer`, Body: CredentialsLegacy("sometoken" /*-------------*/)}, false},
		"manyspaces":          {"Bearer      sometoken" /*--------*/, Credentials{AuthScheme: `Bearer`, Body: CredentialsLegacy("sometoken" /*-------------*/)}, false},
		"params":              {"Bearer foo=bar" /*---------------*/, Credentials{AuthScheme: `Bearer`, Body: CredentialsParameters{{`foo`, `bar`}} /*------*/}, false},
		"illegalcharacter_sp": {"Bearer foo bar" /*---------------*/, Credentials{ /*----------------------------------------------------------------------*/ }, true},
		"illegalcharacter_qm": {"Bearer foo?bar" /*---------------*/, Credentials{ /*----------------------------------------------------------------------*/ }, true},
	}

	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			credentials, err := ParseCredentials(testData.Input)
			crExpected := fmt.Sprintf("%#v", testData.ExpectedCredentials)
			crReceived := fmt.Sprintf("%#v", credentials)
			if crReceived != crExpected {
				t.Errorf("didn't get expected credentials:\n"+
					"\texpected: %s\n"+
					"\treceived: %s",
					crExpected,
					crReceived)
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
