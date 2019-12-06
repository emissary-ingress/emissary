package rfc7235_test

import (
	"fmt"
	"net/http"
	"testing"

	. "github.com/datawire/apro/common/rfc7235"
)

func TestParseChallenges(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		Input              http.Header
		ExpectedChallenges []Challenge
		ExpectedErrors     int
	}{
		"example-rfc7235-4.1": {
			Input: http.Header{
				"WWW-Authenticate": {`Newauth realm="apps", type=1, title="Login to \"apps\"", Basic realm="simple"`},
			},
			ExpectedChallenges: []Challenge{
				{
					AuthScheme: `Newauth`,
					Body: ChallengeParameters{
						{Key: `realm`, Value: `apps`},
						{Key: `type`, Value: `1`},
						{Key: `title`, Value: `Login to "apps"`},
					},
				},
				{
					AuthScheme: `Basic`,
					Body: ChallengeParameters{
						{Key: `realm`, Value: `simple`},
					},
				},
			},
			ExpectedErrors: 0,
		},
		"example-rfc7617-2": {
			Input: http.Header{
				"WWW-Authenticate": {`Basic realm="WallyWorld"`},
			},
			ExpectedChallenges: []Challenge{
				{
					AuthScheme: `Basic`,
					Body: ChallengeParameters{
						{Key: `realm`, Value: `WallyWorld`},
					},
				},
			},
		},
		"example-rfc7617-2.1": {
			Input: http.Header{
				"WWW-Authenticate": {`Basic realm="foo", charset="UTF-8"`},
			},
			ExpectedChallenges: []Challenge{
				{
					AuthScheme: `Basic`,
					Body: ChallengeParameters{
						{Key: `realm`, Value: `foo`},
						{Key: `charset`, Value: `UTF-8`},
					},
				},
			},
		},
		"example-rfc6750-3-1": {
			Input: http.Header{
				"WWW-Authenticate": {`Bearer realm="example"`},
			},
			ExpectedChallenges: []Challenge{
				{
					AuthScheme: `Bearer`,
					Body: ChallengeParameters{
						{Key: `realm`, Value: `example`},
					},
				},
			},
		},
		"example-rfc6750-3-2": {
			Input: http.Header{
				"WWW-Authenticate": {`Bearer realm="example", error="invalid_token", error_description="The access token expired"`},
			},
			ExpectedChallenges: []Challenge{
				{
					AuthScheme: `Bearer`,
					Body: ChallengeParameters{
						{Key: `realm`, Value: `example`},
						{Key: `error`, Value: `invalid_token`},
						{Key: `error_description`, Value: `The access token expired`},
					},
				},
			},
		},
	}

	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			challenges, errors := ParseChallenges("www-authenticate", testData.Input)
			chExpected := fmt.Sprintf("%#v", testData.ExpectedChallenges)
			chReceived := fmt.Sprintf("%#v", challenges)
			if chReceived != chExpected {
				t.Errorf("didn't get expected list of challenges:\n"+
					"\texpected: %s\n"+
					"\treceived: %s",
					chExpected,
					chReceived)
			}
			if len(errors) != testData.ExpectedErrors {
				t.Errorf("expected %d errors, but got %d: %v", testData.ExpectedErrors, len(errors), errors)
			}
		})
	}
}
