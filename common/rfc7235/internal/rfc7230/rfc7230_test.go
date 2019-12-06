package rfc7230_test

import (
	"testing"

	"github.com/datawire/apro/common/rfc7235/internal/rfc7230"
)

func TestScanQuotedString(t *testing.T) {
	testcases := map[string]struct {
		Input         string
		ExpectedValue string
		ExpectedRest  string
		ExpectedError bool
	}{
		"simple":        {`"simple"`, `simple`, ``, false},
		"withRemainder": {`"simple"rest`, `simple`, `rest`, false},
		"noClose":       {`"simple`, ``, ``, true},
		"noOpen":        {`simple"`, ``, ``, true},
		"words":         {`"foo bar" baz`, `foo bar`, ` baz`, false},
		"escapedQuote":  {`"foo\"bar" baz`, `foo"bar`, ` baz`, false},
		"escapedOther":  {`"foo\ bar" baz`, `foo bar`, ` baz`, false},
		"invalidEscape": {"\"foo\\\nbar\" baz", ``, ``, true},
		"illegalOctet":  {"\"\n\"", ``, ``, true},
	}

	t.Parallel()
	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			value, rest, err := rfc7230.ScanQuotedString(testData.Input)
			if value != testData.ExpectedValue {
				t.Errorf("quoted value didn't match what was expected:\n"+
					"\texpected: %q\n"+
					"\treceived: %q",
					testData.ExpectedValue,
					value)
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

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestScanListExample(t *testing.T) {
	// These are the examples given in §7.
	//
	// > For example, given these ABNF productions:
	// >
	// >   example-list      = 1#example-list-elmt
	// >   example-list-elmt = token ; see Section 3.2.6
	// >
	// > Then the following are valid values for example-list (not including
	// > the double quotes, which are present for delimitation only):
	// >
	// >   "foo,bar"
	// >   "foo ,bar,"
	// >   "foo , ,bar,charlie"
	// >
	// > In contrast, the following values would be invalid, since at least
	// > one non-empty element is required by the example-list production:
	// >
	// >   ""
	// >   ","
	// >   ",   ,"
	//
	// Note that the primary document says
	//
	//   "foo , ,bar,charlie   "
	//
	// But Errata ID 4169 corrects it that is invalid (since the
	// list does not include the trailing whitespace), while the
	// the same string without the trailing whitespace _is_ valid.

	scanExampleList := func(input string) ([]string, string, error) {
		untypedRet, rest, err := rfc7230.ScanList(input, 1, 0, func(input string) (interface{}, string, error) { return rfc7230.ScanToken(input) })
		if err != nil {
			return nil, "", err
		}
		ret := make([]string, 0, len(untypedRet))
		for _, el := range untypedRet {
			ret = append(ret, el.(string))
		}
		return ret, rest, nil
	}

	testcases := map[string]struct {
		Input         string
		ExpectedList  []string
		ExpectedRest  string
		ExpectedError bool
	}{
		"valid-1":             {"foo,bar" /*---------------*/, []string{"foo", "bar"}, "", false},
		"valid-2":             {"foo ,bar," /*-------------*/, []string{"foo", "bar"}, "", false},
		"valid-3":             {"foo , ,bar,charlie" /*----*/, []string{"foo", "bar", "charlie"}, "", false},
		"valid-3-with-suffix": {"foo , ,bar,charlie   " /*-*/, []string{"foo", "bar", "charlie"}, "   ", false},
		"invalid-1":           {"" /*----------------------*/, nil, "", true},
		"invalid-2":           {"," /*---------------------*/, nil, "", true},
		"invalid-3":           {",   ," /*-----------------*/, nil, "", true},
	}

	t.Parallel()
	for testName, testData := range testcases {
		testData := testData // capture loop variable
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			list, rest, err := scanExampleList(testData.Input)
			if !stringSliceEqual(list, testData.ExpectedList) {
				t.Errorf("list did not match expected value:\n"+
					"\texpected: %#v\n"+
					"\treceived: %#v",
					testData.ExpectedList,
					list)
			}
			if rest != testData.ExpectedRest {
				t.Errorf("rest did not match expected value:\n"+
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
