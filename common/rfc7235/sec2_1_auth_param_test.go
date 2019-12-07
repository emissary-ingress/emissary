package rfc7235

import (
	"testing"
)

func TestAuthParamString(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		Input    AuthParam
		Expected string
	}{
		"simple":      {Input: AuthParam{Key: "foo", Value: "bar"}, Expected: `foo=bar`},
		"realm":       {Input: AuthParam{Key: "realm", Value: "bar"}, Expected: `realm="bar"`},
		"needsQuote":  {Input: AuthParam{Key: "foo", Value: "bar baz"}, Expected: `foo="bar baz"`},
		"escapeQuote": {Input: AuthParam{Key: "foo", Value: `bar="baz"`}, Expected: `foo="bar=\"baz\""`},
	}

	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			if received := testData.Input.String(); received != testData.Expected {
				t.Errorf("param didn't stringify as expected:\n"+
					"\texpected: %q\n"+
					"\treceived: %q",
					testData.Expected,
					received)
			}
		})
	}
}

func TestAuthParamScan(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		Input         string
		ExpectedParam AuthParam
		ExpectedRest  string
		ExpectedError bool
	}{
		"docExample":    {`foo = "bar baz" remainder`, AuthParam{Key: `foo`, Value: `bar baz`}, ` remainder`, false},
		"simple":        {`foo = bar remainder`, AuthParam{Key: `foo`, Value: `bar`}, ` remainder`, false},
		"noRemainder":   {`foo = "bar baz"`, AuthParam{Key: `foo`, Value: `bar baz`}, ``, false},
		"quoted":        {`foo = "bar"  remainder`, AuthParam{Key: `foo`, Value: `bar`}, `  remainder`, false},
		"escapedQuote":  {`foo = "bar\"baz" remainder`, AuthParam{Key: `foo`, Value: `bar"baz`}, ` remainder`, false},
		"escapedOther":  {`foo = "bar\ baz" remainder`, AuthParam{Key: `foo`, Value: `bar baz`}, ` remainder`, false},
		"noCloseQuote":  {`foo = "bar remainder`, AuthParam{}, "", true},
		"invalidEscape": {"foo=\"bar\\\nbaz\"", AuthParam{}, "", true},
		"illegalOctet":  {"foo=\"\n\"", AuthParam{}, "", true},
		"specExample":   {`realm="simple"`, AuthParam{Key: `realm`, Value: `simple`}, ``, false},
	}
	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			param, rest, err := scanAuthParam(testData.Input)
			if param != testData.ExpectedParam {
				t.Errorf("param didn't parse as expected:\n"+
					"\texpected: %#v\n"+
					"\treceived: %#v",
					testData.ExpectedParam,
					param)
			}
			if rest != testData.ExpectedRest {
				t.Errorf("unparsed input didn't match what was expected:\n"+
					"\texpected: %q\n"+
					"\treceived: %q",
					testData.ExpectedRest,
					rest)
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
