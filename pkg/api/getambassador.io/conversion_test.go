package getambassadorio

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

func marshalNormalized(t *testing.T, typed interface{}) string {
	t.Helper()

	// First we discard any type information by seralizing-then-deserializing.  This is
	// important so that the order of struct fields doesn't impact the normalized serialization.

	bs, err := json.Marshal(typed)
	require.NoError(t, err)

	var untyped interface{}
	err = json.Unmarshal(bs, &untyped)
	require.NoError(t, err)

	// Now serialize for real.

	out, err := json.MarshalIndent(untyped, "", "\t")
	require.NoError(t, err)

	return string(out)
}

func requireEqualNormalized(t *testing.T, exp, act interface{}) {
	t.Helper()
	expStr := marshalNormalized(t, exp)
	actStr := marshalNormalized(t, act)
	// Do this directly instead of using require.Equal so that the "%q" version doesn't spam
	// stdout.
	if expStr != actStr {
		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expStr),
			B:        difflib.SplitLines(actStr),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  3,
		})
		t.Errorf("Not equal\n%s", diff)
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()
	testcases := map[string]map[string]interface{}{
		"authsvc": {
			"getambassador.io/v2":       v2.AuthService{},
			"getambassador.io/v3alpha1": v3alpha1.AuthService{},
		},
		"devportals": {
			"getambassador.io/v2":       v2.DevPortal{},
			"getambassador.io/v3alpha1": v3alpha1.DevPortal{},
		},
		"hosts": {
			"getambassador.io/v2":       v2.Host{},
			"getambassador.io/v3alpha1": v3alpha1.Host{},
		},
		"logsvc": {
			"getambassador.io/v2":       v2.LogService{},
			"getambassador.io/v3alpha1": v3alpha1.LogService{},
		},
		"mappings": {
			"getambassador.io/v2":       v2.Mapping{},
			"getambassador.io/v3alpha1": v3alpha1.Mapping{},
		},
		"modules": {
			"getambassador.io/v2":       v2.Module{},
			"getambassador.io/v3alpha1": v3alpha1.Module{},
		},
		"ratelimitsvc": {
			"getambassador.io/v2":       v2.RateLimitService{},
			"getambassador.io/v3alpha1": v3alpha1.RateLimitService{},
		},
		"tcpmappings": {
			"getambassador.io/v2":       v2.TCPMapping{},
			"getambassador.io/v3alpha1": v3alpha1.TCPMapping{},
		},
		"tlscontexts": {
			"getambassador.io/v2":       v2.TLSContext{},
			"getambassador.io/v3alpha1": v3alpha1.TLSContext{},
		},
		"tracingsvc": {
			"getambassador.io/v2":       v2.TracingService{},
			"getambassador.io/v3alpha1": v3alpha1.TracingService{},
		},
	}

	scheme := BuildScheme()

	t.Run("RoundTrip", func(t *testing.T) {
		t.Parallel()
		for typename := range testcases {
			typename := typename
			t.Run(typename, func(t *testing.T) {
				t.Parallel()
				for mainAPIVersion := range testcases[typename] {
					for throughAPIVersion := range testcases[typename] {
						if mainAPIVersion == throughAPIVersion {
							continue
						}
						mainAPIVersion := mainAPIVersion
						throughAPIVersion := throughAPIVersion
						testname := path.Base(mainAPIVersion) + "_through_" + path.Base(throughAPIVersion)
						t.Run(testname, func(t *testing.T) {
							t.Parallel()
							inBytes, err := ioutil.ReadFile(filepath.Join(path.Base(mainAPIVersion), "testdata", typename+".yaml"))
							require.NoError(t, err)
							inListPtr := reflect.New(reflect.SliceOf(reflect.TypeOf(testcases[typename][mainAPIVersion])))
							require.NoError(t, yaml.Unmarshal(inBytes, inListPtr.Interface()))
							inList := inListPtr.Elem()
							listLen := inList.Len()

							midList := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(testcases[typename][throughAPIVersion])), listLen, listLen)
							for i := 0; i < listLen; i++ {
								require.NoError(t, scheme.Convert(inList.Index(i).Addr().Interface(), midList.Index(i).Addr().Interface(), v2.DisableManglingAmbassadorID{}))
							}

							outList := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(testcases[typename][mainAPIVersion])), listLen, listLen)
							for i := 0; i < listLen; i++ {
								require.NoError(t, scheme.Convert(midList.Index(i).Addr().Interface(), outList.Index(i).Addr().Interface(), v2.DisableManglingAmbassadorID{}))
								outList.Index(i).FieldByName("TypeMeta").Set(inList.Index(i).FieldByName("TypeMeta"))
							}

							requireEqualNormalized(t, inList.Interface(), outList.Interface())
						})
					}
				}
			})
		}
	})
}
