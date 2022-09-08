package ambex

import (
	"fmt"
	"os"
	"testing"

	cachingv1 "github.com/emissary-ingress/emissary/v3/internal/ir/caching/v1"
	"github.com/emissary-ingress/emissary/v3/internal/ir/types"
	listenerv3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/stretchr/testify/assert"
)

func TestInjectCacheFilter(t *testing.T) {

	type testCase struct {
		TestName      string
		CachePolices  []cachingv1.CachePolicyContext
		CacheMap      cachingv1.CacheMap
		ErrorExpected bool
		ErrorMessage  string
	}

	testcases := []testCase{
		{
			TestName:     "no-filters-configured",
			CachePolices: []cachingv1.CachePolicyContext{},
			CacheMap:     cachingv1.CacheMap{},
		},
		{
			TestName:     "no-hcm-filter",
			CachePolices: []cachingv1.CachePolicyContext{},
			CacheMap:     cachingv1.CacheMap{},
		},
		{
			TestName: "no-policy-matches",
			CachePolices: []cachingv1.CachePolicyContext{
				{
					NamespacedName: types.NamespacedName{Namespace: "default", Name: "simple-cache-policy"},
					Rules: []cachingv1.CacheRuleContext{
						{Host: "bar.com", Path: "/", CacheRef: types.NamespacedName{Namespace: "default", Name: "simple-cache"}},
					},
				},
			},
			CacheMap: cachingv1.CacheMap{
				"default/simple-cache": {
					NamespacedName: types.NamespacedName{Namespace: "default", Name: "simple-cache"},
					ProviderType:   cachingv1.InMemoryCacheProvider,
				},
			},
		},
		{
			TestName: "some-policy-matches",
			CachePolices: []cachingv1.CachePolicyContext{
				{
					NamespacedName: types.NamespacedName{Namespace: "default", Name: "simple-cache-policy"},
					Rules: []cachingv1.CacheRuleContext{
						{Host: "bar.com", Path: "/", CacheRef: types.NamespacedName{Namespace: "default", Name: "simple-cache"}},
					},
				},
			},
			CacheMap: cachingv1.CacheMap{
				"default/simple-cache": {
					NamespacedName: types.NamespacedName{Namespace: "default", Name: "simple-cache"},
					ProviderType:   cachingv1.InMemoryCacheProvider,
				},
			},
		},
		{
			TestName: "missing-cache",
			CachePolices: []cachingv1.CachePolicyContext{
				{
					NamespacedName: types.NamespacedName{Namespace: "default", Name: "simple-cache-policy"},
					Rules: []cachingv1.CacheRuleContext{
						{Host: "bar.com", Path: "/", CacheRef: types.NamespacedName{Namespace: "default", Name: "simple-cache"}},
					},
				},
			},
			CacheMap: cachingv1.CacheMap{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.TestName, func(t *testing.T) {

			inputData, err := os.ReadFile(fmt.Sprintf("testdata/inject-cache-filter/%s.in.json", tc.TestName))
			assert.NoError(t, err)

			input := &listenerv3.Listener{}
			err = protojson.Unmarshal(inputData, input)
			assert.NoError(t, err)

			expectedData, err := os.ReadFile(fmt.Sprintf("testdata/inject-cache-filter/%s.out.json", tc.TestName))
			assert.NoError(t, err)

			expected := &listenerv3.Listener{}
			err = protojson.Unmarshal(expectedData, expected)
			assert.NoError(t, err)

			actual, err := injectCacheFilter(input, tc.CachePolices, tc.CacheMap)

			if tc.ErrorExpected {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.ErrorMessage)
			} else {
				assert.NoError(t, err)

				assert.True(t, cmp.Equal(expected, actual, protocmp.Transform()), cmp.Diff(expected, actual, protocmp.Transform()))
			}

		})
	}
}
