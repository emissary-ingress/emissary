package licensekeys_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/datawire/apro/lib/licensekeys"
)

var allFeatures = []licensekeys.Feature{
	licensekeys.FeatureUnrecognized,
	licensekeys.FeatureTraffic,
	licensekeys.FeatureRateLimit,
	licensekeys.FeatureFilter,
	licensekeys.FeatureDevPortal,
}

func featureInArray(needle licensekeys.Feature, haystack []licensekeys.Feature) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

func TestRequireFeature(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		Key      string
		Err      *regexp.Regexp
		Features []licensekeys.Feature
	}{
		"empty": {
			Key: "",
			Err: regexp.MustCompile("^Token validation error: .*[Mm]alformed"),
		},
		"malformed": {
			Key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			Err: regexp.MustCompile("^Token validation error: .*[Mm]alformed"),
		},
		"bad signature": {
			Key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8bogon",
			Err: regexp.MustCompile("^Token validation error: .*signature is invalid"),
		},
		"v0": {
			Key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8",
			Features: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
				licensekeys.FeatureRateLimit,
				licensekeys.FeatureFilter,
			},
		},
		"v0 expired": {
			// Created with `./bin/apictl-key create --id=dev --expiration=0` from commit f2458be558c307a2787cdd0434fbbd3f86c617e7
			Key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6MTU2NjIzMjU1MX0.ihYqQ9w_vIUtm_dl1FCH7oAMDFsSitr1yCiGhjYCTdc",
			Err: regexp.MustCompile("^Token validation error: .*[Ee]xpired"),
		},
		"v1 default": {
			// Created with `./bin/apictl-key create --id=dev --expiration=$((100*365))`
			Key: "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyJdLCJleHAiOjQ3MTk4MzA5NTksImlhdCI6MTU2NjIzMDk1OSwibmJmIjoxNTY2MjMwOTU5fQ.DJ-BzcW2nBxiAmLc1LnbBhh5YfQ32W4bnRaIq07N1yYZjNp-EQDLiMJY_158x34u6tJN1iXl4HY0aaeYGzTgMk2HjW14rEv5Lt4M_DrQub_UQB9TbFjTdNtlBxJ-EI8l-CB_IO3jIYz4TwNUIk90kXhlBFk-CXw3aWZ8FLk0k_oSzzw414HwQRmi1walh1fqywOYg6v6kRTE2QDWJmWSeYjx4kXIh79gYaOhxsxAC_XtIO4KQIC7NUIKcT62EfCbZ4lT6zJgfcnuCLrZSiAj2131p2YQ3VKj8r1V5jtb76oKORItdarEX-isYuJKvg48UzTuucIT-8c6V91PRKtQvQ",
			Features: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
				licensekeys.FeatureRateLimit,
				licensekeys.FeatureFilter,
			},
		},
		"v1 with devportal": {
			// Created with `./bin/apictl-key create --id=dev --expiration=$((100*365)) --features=filter,ratelimit,traffic,devportal`
			Key: "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCJdLCJleHAiOjQ3MTg4ODYzNDYsImlhdCI6MTU2NTI4NjM0NiwibmJmIjoxNTY1Mjg2MzQ2fQ.i0Uf2pkTGJfniL5K0YTLk3vwJO6JPvTeDcCRzDc3tE-ZDK37zg3yVq46QFOWPxzfDFA-GQFlNCiZWvHI45XH4fxvb5A_hMScykeXraJd0LRqugbtWQuh1LVf9tgx08GZ0q5rIo_fRY04D0UMbbl7a6hRYJ7FkSlUzIzKVXqBwF0wJrLJVh5gD_PxyqbD1uGmS9v-i3T2vr4yHA7MPR0TR5XGRZCIYhfiZ8bHszCbzPYC5EPSYLF2oTmlm6y4xWSQKz9Grm1IhYN4mSynM7n5oY9y1Be2iPwUhU6yzfRPnOCbFBAp1h6wS6WJOyWrdefAzn3oVccwNMZSKAs4aYbEqA",
			Features: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
				licensekeys.FeatureRateLimit,
				licensekeys.FeatureFilter,
				licensekeys.FeatureDevPortal,
			},
		},
		"v1 with just filter": {
			// Created with `./bin/apictl-key create --id=dev --expiration=$((100*365)) --features=filter`
			Key: "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIl0sImV4cCI6NDcxOTgzMjM4NCwiaWF0IjoxNTY2MjMyMzg0LCJuYmYiOjE1NjYyMzIzODR9.BPAPuZtz6_T0zxjfPGHtQXP9F_Abvr9jbWQvyQjAFr8N0fkYEXp13g9ctaem5orvghoT72yrEl6SX6GUGuV5RfNs-VMp05AvkooCX5ndvSA6h_hLG8pz4wATFRCW8cuU04rfCdn8xs5EzNPPgiBDs2vyL9yxDdcLDGkXAuaVRvEkScdJDsCewlBnVi_TrItE7HyQVG9tcW0rcLbaLBySa1TpiRy1G9vjDaZgUftI5ywcS44TK72i2zt4I8uDueZUw7t8815c00wFv1HQ4Pu9OVObr0h7DvTai4yfClhebRDv4Z3lQMrsv9n2KYv9JLVqrfstqhBKCrclPb78hAKi3g",
			Features: []licensekeys.Feature{
				licensekeys.FeatureFilter,
			},
		},
		"v1 expired": {
			// Created with `./bin/apictl-key create --id=dev --expiration=0 --features=filter,ratelimit,traffic,devportal`
			Key: "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCJdLCJleHAiOjE1NjYyMjY3MzcsImlhdCI6MTU2NjIyNjczNywibmJmIjoxNTY2MjI2NzM3fQ.IZezOL2ocqXuWRsOu545wh62esJxRht85wqbpwD8weWSeu9-K7benJKEV5t1xpUqP2OGzBjXO4KNagb8kDu1NA8rqVr87VvcsSFDvM0emCg6vREZqcLcMy65olo-HaNtDi5TFq4eQvQw3UdbsqCixOhbCFReeG7XdqTuEzbCbKmx8dLutjQKzTrILYWzCF_sGXhue-OcQGo1NbZS5X1DLypu2vPqFSdnGb47dMY2N4MewKwUMsrs8SOeFVnNsU9jEMCea1BsMPJsOycZAuoYsVVOa17KeAGTpR2zzH0w5TbXeOTnsyk0WChj206AjS5rBpgf3byyhcgRgv7wzrQ5SQ",
			Err: regexp.MustCompile("^Token validation error: .*[Ee]xpired"),
		},
		"v1 with v0 signing key": {
			// The point of this test is that even if an adversary does figure out the v0 signing key
			// ("1234", the same as combination on my luggage), they can't get access to newer features that
			// require a v1 license key.
			//
			// Created by modifying apictl-key to use the v0 signing stuff, and running
			// `./bin/apictl-key create --id=dev --expiration=$((100*365)) --features=filter,ratelimit,traffic,devportal`
			Key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCJdLCJleHAiOjQ3MTk4MzE5MjgsImlhdCI6MTU2NjIzMTkyOCwibmJmIjoxNTY2MjMxOTI4fQ.r2uSJrnnYt_1a7HGPlPZub8FNZQU2ZFZbLvl7QH4R7I",
			Err: regexp.MustCompile("^Token validation error: signing method HS256 is invalid"),
		},
		"v1 with unknown features": {
			// The point of this test is that older APro don't reject license keys that enabled more recent
			// (unknown to older versions) features.
			//
			// Created by  modifying apictl-key to accept unknown --features, and running
			// `./bin/apictl-key create --id=dev --expiration=$((100*365)) --features=devportal,bogus,borked`
			Key: "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZGV2cG9ydGFsIiwiYm9ndXMiLCJib3JrZWQiXSwiZXhwIjo0NzE5ODMyMDk1LCJpYXQiOjE1NjYyMzIwOTUsIm5iZiI6MTU2NjIzMjA5NX0.ThtzBpq1ybe2cTV98ZwqMs3pakgCl9vzXsony_FA9cIXRjzWxbyt9e4pWpskM132J26ACYmaGvshEYGutsL5QFVD4LF6wz9bAPbWDpCFlrzl1QevDFvXWoPCwBBSlK-RPs_8xChuK9Wpw273fYMRI_49j4Ml6EEl6uHeqKO58wUkeVF_eoDReZDrJFpL2Uenm85mGqkmHjOpHFBNMCHNXl_HwHYOj-Jnob85Ya79yU3HotVegKKVTxPQL-il5EqekooQBhJ-ELd7dm7-ubSiqN472QuqO0BAOG0x36ENz3hKo2OhqCdlSDcbLgNfw7b_iIO2gB1ySK2gA921-ICjfg",
			Features: []licensekeys.Feature{
				licensekeys.FeatureDevPortal,
				licensekeys.FeatureUnrecognized,
			},
		},
	}
	for tcName, tc := range tcs {
		tc := tc // capture loop variable
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			// Sanity-check the testcase itself
			for _, feature := range tc.Features {
				if !featureInArray(feature, allFeatures) {
					t.Errorf("error: test case included feature %v that is not in allFeatures array", feature)
				}
			}
			if tc.Err != nil && len(tc.Features) > 0 {
				t.Errorf("error: test case included functional features, despite expecting an error")
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
			for _, feature := range allFeatures {
				err := claims.RequireFeature(feature)
				t.Logf("info : claims.RequireFeature(%q) = %v", feature, err)
				if featureInArray(feature, tc.Features) != (err == nil) {
					if err == nil {
						t.Errorf("error: license key should not grant feature=%q, but it did", feature)
					} else {
						t.Errorf("error: license key should grant feature=%q, but it did not: %v", feature, err)
					}
				}
			}
		})
	}
}

func TestParseFeature(t *testing.T) {
	t.Parallel()
	for _, feature := range allFeatures {
		parsed, parsedOK := licensekeys.ParseFeature(feature.String())
		if !parsedOK {
			t.Errorf("could not parse %#v.String()", feature)
		} else if parsed != feature {
			t.Errorf("round-trip licensekeys.ParseFeature(%#v.String()) failed; got back %#v",
				feature, parsed)
		}
	}
}

func TestListFeatures(t *testing.T) {
	t.Parallel()
	for _, featureStr := range licensekeys.ListKnownFeatures() {
		parsed, parsedOK := licensekeys.ParseFeature(featureStr)
		switch {
		case !parsedOK:
			t.Errorf("could not parse %q", featureStr)
		case parsed.String() != featureStr:
			t.Errorf("round-trip licensekeys.ParseFeature(%q).String() failed; got back %q",
				featureStr, parsed.String())
		case !featureInArray(parsed, allFeatures):
			t.Errorf("unexpected feature from .ListKnownFeatures(): %q", featureStr)
		}
	}
}

func TestFeatureUnmarshalJSON(t *testing.T) {
	t.Parallel()
	expected := licensekeys.FeatureFilter
	var feature licensekeys.Feature
	if err := json.Unmarshal([]byte(`"filter"`), &feature); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if feature != expected {
		t.Fatalf("json.Unmarshal gave unexpected result:\n\texpected: %#v\n\treceived: %#v",
			expected, feature)
	}
}

func TestFeatureMarshalJSON(t *testing.T) {
	t.Parallel()
	expected := `"filter"`
	jsonBytes, err := json.Marshal(licensekeys.FeatureFilter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(jsonBytes) != expected {
		t.Fatalf("json.Marshal gave unexpected result:\n\texpected: %#v\n\treceived: %#v",
			expected, string(jsonBytes))
	}
}
