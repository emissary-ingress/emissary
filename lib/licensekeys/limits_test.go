package licensekeys_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/datawire/apro/lib/licensekeys"
)

var allLimits = []licensekeys.Limit{
	licensekeys.LimitUnrecognized,
	licensekeys.LimitDevPortalServices,
}

func limitInArray(needle licensekeys.Limit, haystack []licensekeys.Limit) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

func limitInValueArray(needle licensekeys.Limit, haystack []licensekeys.LimitValue) bool {
	for _, straw := range haystack {
		if straw.Name == needle {
			return true
		}
	}
	return false
}

func limitValueInArray(needle licensekeys.Limit, haystack []licensekeys.LimitValue) int {
	for _, straw := range haystack {
		if straw.Name == needle {
			return straw.Value
		}
	}
	return licensekeys.GetLimitDefault(needle)
}

func TestGetLimitValue(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		Key    string
		Err    *regexp.Regexp
		Limits []licensekeys.LimitValue
	}{
		"v1 default": {
			// Created with `./bin/apictl-key create --id=dev --expiration=$((100*365))`
			Key:    "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyJdLCJleHAiOjQ3MTk4MzA5NTksImlhdCI6MTU2NjIzMDk1OSwibmJmIjoxNTY2MjMwOTU5fQ.DJ-BzcW2nBxiAmLc1LnbBhh5YfQ32W4bnRaIq07N1yYZjNp-EQDLiMJY_158x34u6tJN1iXl4HY0aaeYGzTgMk2HjW14rEv5Lt4M_DrQub_UQB9TbFjTdNtlBxJ-EI8l-CB_IO3jIYz4TwNUIk90kXhlBFk-CXw3aWZ8FLk0k_oSzzw414HwQRmi1walh1fqywOYg6v6kRTE2QDWJmWSeYjx4kXIh79gYaOhxsxAC_XtIO4KQIC7NUIKcT62EfCbZ4lT6zJgfcnuCLrZSiAj2131p2YQ3VKj8r1V5jtb76oKORItdarEX-isYuJKvg48UzTuucIT-8c6V91PRKtQvQ",
			Limits: []licensekeys.LimitValue{},
		},
		"v1 with unknown limits": {
			// The point of this test is that older APro don't reject license keys that enabled more recent
			// (unknown to older versions) limits.
			//
			// Created by  modifying apictl-key to accept unknown --limits, and running
			// `./bin/apictl-key create --id=dev --expiration=$((100*365)) --limits=devportal,bogus,borked`
			Key:    "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZGV2cG9ydGFsIiwiYm9ndXMiLCJib3JrZWQiXSwiZXhwIjo0NzE5ODMyMDk1LCJpYXQiOjE1NjYyMzIwOTUsIm5iZiI6MTU2NjIzMjA5NX0.ThtzBpq1ybe2cTV98ZwqMs3pakgCl9vzXsony_FA9cIXRjzWxbyt9e4pWpskM132J26ACYmaGvshEYGutsL5QFVD4LF6wz9bAPbWDpCFlrzl1QevDFvXWoPCwBBSlK-RPs_8xChuK9Wpw273fYMRI_49j4Ml6EEl6uHeqKO58wUkeVF_eoDReZDrJFpL2Uenm85mGqkmHjOpHFBNMCHNXl_HwHYOj-Jnob85Ya79yU3HotVegKKVTxPQL-il5EqekooQBhJ-ELd7dm7-ubSiqN472QuqO0BAOG0x36ENz3hKo2OhqCdlSDcbLgNfw7b_iIO2gB1ySK2gA921-ICjfg",
			Limits: []licensekeys.LimitValue{},
		},
	}
	for tcName, tc := range tcs {
		tc := tc // capture loop variable
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			// Sanity-check the testcase itself
			for _, limit := range tc.Limits {
				if !limitInArray(limit.Name, allLimits) {
					t.Errorf("error: test case included limit %v that is not in allLimits array", limit)
				}
			}
			if tc.Err != nil && len(tc.Limits) > 0 {
				t.Errorf("error: test case included functional limits, despite expecting an error")
			}
			if t.Failed() {
				return
			}
			// OK, now actually run the test case
			claims, err := licensekeys.ParseKey(tc.Key)
			if (claims == nil) == (err == nil) {
				t.Fatalf("error: exactly one of 'claims' or 'err' should be nil, not 0 or both:\n\tclaims = %v\n\terr = %v", claims, err)
			}
			switch {
			case err == nil && tc.Err == nil:
				// continue
			case err == nil && tc.Err != nil:
				t.Fatal("error: expected an error, but got err == nil")
			case err != nil && tc.Err != nil:
				if !tc.Err.MatchString(err.Error()) {
					t.Fatalf("error: expected an error, and got an error, but not the expected error\n\texpected: regex = %s\n\treceived: %v",
						tc.Err.String(), err)
				}
				// the test has now passed
				return
			case err != nil && tc.Err == nil:
				t.Fatalf("error: exected no error, but got err = %v", err)
			}
			for _, limit := range allLimits {
				val := claims.GetLimitValue(limit)
				expected := limitValueInArray(limit, tc.Limits)
				t.Logf("info : claims.GetLimitValue(%q) = %v", limit, val)
				if val != expected {
					if limitInValueArray(limit, tc.Limits) {
						t.Errorf("error: license key should grant limit=%q value: %v, but it granted: %v", limit, expected, val)
					} else {
						t.Errorf("error: license key should grant limit=%q default value %v, but it granted %v", limit, expected, val)
					}
				}
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	for _, limit := range allLimits {
		parsed, parsedOK := licensekeys.ParseLimit(limit.String())
		if !parsedOK {
			t.Errorf("could not parse %#v.String()", limit)
		} else if parsed != limit {
			t.Errorf("round-trip licensekeys.ParseLimit(%#v.String()) failed; got back %#v",
				limit, parsed)
		}
	}
}

func TestParseLimitValue(t *testing.T) {
	t.Parallel()
	for i, limit := range allLimits {
		expected := licensekeys.LimitValue{Name: limit, Value: 1234500 + i}
		str := expected.String()
		parsed, err := licensekeys.ParseLimitValue(str)
		if err != nil {
			t.Errorf("could not parse %q: %v", str, err)
		} else if parsed != expected {
			t.Errorf("round-trip licensekeys.ParseLimitValue(%#v.String()) failed; got back %#v",
				expected, parsed)
		}
	}
	foo := func(value string) string {
		return licensekeys.LimitDevPortalServices.String() + "=" + value
	}
	for _, str := range []string{"very unlikely limit name", foo(""), foo("NotAnInt")} {
		_, err := licensekeys.ParseLimitValue(str)
		if err == nil {
			t.Errorf("Should not parse %q", str)
		}
	}
}

func TestListLimits(t *testing.T) {
	t.Parallel()
	for _, limitStr := range licensekeys.ListKnownLimits() {
		parsed, parsedOK := licensekeys.ParseLimit(limitStr)
		switch {
		case !parsedOK:
			t.Errorf("could not parse %q", limitStr)
		case parsed.String() != limitStr:
			t.Errorf("round-trip licensekeys.ParseLimit(%q).String() failed; got back %q",
				limitStr, parsed.String())
		case !limitInArray(parsed, allLimits):
			t.Errorf("unexpected limit from .ListKnownLimits(): %q", limitStr)
		}
	}
}

func TestLimitUnmarshalJSON(t *testing.T) {
	t.Parallel()
	expected := licensekeys.LimitValue{
		Name:  licensekeys.LimitDevPortalServices,
		Value: 17,
	}
	var limit licensekeys.LimitValue
	if err := json.Unmarshal([]byte(`{"l": "devportal-services", "v": 17}`), &limit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != expected {
		t.Fatalf("json.Unmarshal gave unexpected result:\n\texpected: %#v\n\treceived: %#v",
			expected, limit)
	}
}

func TestLimitMarshalJSON(t *testing.T) {
	t.Parallel()
	expected := `{"l":"devportal-services","v":42}`
	jsonBytes, err := json.Marshal(licensekeys.LimitValue{
		Name:  licensekeys.LimitDevPortalServices,
		Value: 42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(jsonBytes) != expected {
		t.Fatalf("json.Marshal gave unexpected result:\n\texpected: %#v\n\treceived: %#v",
			expected, string(jsonBytes))
	}
}
