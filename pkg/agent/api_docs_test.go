package agent_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/pkg/agent"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

func TestAPIDocsStore(t *testing.T) {
	shouldIgnore := true
	mappingRewrite := "/internal-path"
	headerValue := "header-value"

	type testCases struct {
		name     string
		mappings []*v3alpha1.Mapping

		rawJSONDocsContent string
		JSONDocsErr        error

		expectedRequestURL     string
		expectedRequestHost    string
		expectedRequestHeaders []agent.Header
		expectedSOTW           []*snapshotTypes.APIDoc
	}
	cases := []*testCases{
		{
			name: "will ignore mappings without a 'docs' property",
			mappings: []*v3alpha1.Mapping{
				{
					Spec: v3alpha1.MappingSpec{
						Prefix:  "",
						Service: "some-svc",
						Docs:    nil,
					},
					ObjectMeta: kates.ObjectMeta{
						Name:      "some-endpoint",
						Namespace: "default",
					},
				},
				nil,
			},
			expectedSOTW: []*snapshotTypes.APIDoc{},
		},
		{
			name: "will ignore mappings with docs.ignored setting",
			mappings: []*v3alpha1.Mapping{
				{
					Spec: v3alpha1.MappingSpec{
						Prefix:  "",
						Service: "some-svc",
						Docs: &v3alpha1.DocsInfo{
							Ignored: &shouldIgnore,
						},
					},
					ObjectMeta: kates.ObjectMeta{
						Name:      "some-endpoint",
						Namespace: "default",
					},
				},
			},
			expectedSOTW: []*snapshotTypes.APIDoc{},
		},
		{
			name: "will scrape OpenAPI docs from docs path and ignore malformed OpenAPI docs",
			mappings: []*v3alpha1.Mapping{
				{
					Spec: v3alpha1.MappingSpec{
						Prefix:  "",
						Rewrite: &mappingRewrite,
						Service: "https://some-svc.fqdn:443",
						Docs: &v3alpha1.DocsInfo{
							DisplayName: "docs-display-name",
							Path:        "/docs-location",
						},
						Headers: map[string]v3alpha1.BoolOrString{
							"header-key": {
								String: &headerValue,
							},
						},
					},
					ObjectMeta: kates.ObjectMeta{
						Name:      "some-endpoint",
						Namespace: "default",
					},
				},
			},

			rawJSONDocsContent: "this is not JSON",

			expectedRequestURL:     "https://some-svc.fqdn:443/internal-path/docs-location",
			expectedRequestHost:    "",
			expectedRequestHeaders: []agent.Header{{Name: "header-key", Value: "header-value"}},
			expectedSOTW:           []*snapshotTypes.APIDoc{},
		},
		{
			name: "will scrape OpenAPI docs from docs path",
			mappings: []*v3alpha1.Mapping{
				{
					TypeMeta: kates.TypeMeta{
						Kind: "Mapping",
					},
					Spec: v3alpha1.MappingSpec{
						Prefix:  "/prefix",
						Service: "some-svc:8080",
						Docs: &v3alpha1.DocsInfo{
							DisplayName: "docs-display-name",
							Path:        "/docs-location",
						},
						DeprecatedHost: "mapping-host",
						Hostname:       "mapping-hostname",
					},
					ObjectMeta: kates.ObjectMeta{
						Name:      "some-endpoint",
						Namespace: "default",
					},
				},
			},

			rawJSONDocsContent: `{"openapi":"3.0.0", "info":{"title": "Sample API", "version":"0.0"}, "paths":{}}`,

			expectedRequestURL:     "http://some-svc.default:8080/docs-location",
			expectedRequestHost:    "mapping-hostname",
			expectedRequestHeaders: []agent.Header{},
			expectedSOTW: []*snapshotTypes.APIDoc{{
				TypeMeta: &kates.TypeMeta{
					Kind:       "OpenAPI",
					APIVersion: "v3",
				},
				Metadata: &kates.ObjectMeta{
					Name: "docs-display-name",
				},
				TargetRef: &kates.ObjectReference{
					Kind:      "Mapping",
					Name:      "some-endpoint",
					Namespace: "default",
				},
				Data: []byte(`{"components":{},"info":{"title":"Sample API","version":"0.0"},"openapi":"3.0.0","paths":{},"servers":[{"url":"mapping-hostname/prefix"}]}`),
			}},
		},
		{
			name: "will scrape OpenAPI docs from docs url",
			mappings: []*v3alpha1.Mapping{
				{
					TypeMeta: kates.TypeMeta{
						Kind: "Mapping",
					},
					Spec: v3alpha1.MappingSpec{
						Prefix:  "/api-prefix",
						Service: "some-svc",
						Docs: &v3alpha1.DocsInfo{
							URL: "https://external-url",
						},
						Hostname: "*",
					},
					ObjectMeta: kates.ObjectMeta{
						Name:      "some-endpoint",
						Namespace: "default",
					},
				},
			},

			rawJSONDocsContent: `{"openapi":"3.0.0", "info":{"title": "Sample API", "version":"0.0"}, "paths":{}}`,

			expectedRequestURL:     "https://external-url",
			expectedRequestHost:    "",
			expectedRequestHeaders: []agent.Header{},
			expectedSOTW: []*snapshotTypes.APIDoc{{
				TypeMeta: &kates.TypeMeta{
					Kind:       "OpenAPI",
					APIVersion: "v3",
				},
				Metadata: &kates.ObjectMeta{
					Name: "some-endpoint.default",
				},
				TargetRef: &kates.ObjectReference{
					Kind:      "Mapping",
					Name:      "some-endpoint",
					Namespace: "default",
				},
				Data: []byte(`{"components":{},"info":{"title":"Sample API","version":"0.0"},"openapi":"3.0.0","paths":{},"servers":[{"url":""}]}`),
			}},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(dlog.NewTestContext(t, false))
			defer cancel()

			snapshot := &snapshotTypes.Snapshot{
				Kubernetes: &snapshotTypes.KubernetesSnapshot{
					Mappings: c.mappings,
				},
			}

			store := agent.NewAPIDocsStore()
			store.Client = NewMockAPIDocsHTTPClient(t, c.expectedRequestURL, c.expectedRequestHost, c.expectedRequestHeaders, c.rawJSONDocsContent, c.JSONDocsErr)

			// Processing the test case snapshot should yield the expected state of the world
			store.ProcessSnapshot(ctx, snapshot)
			assert.Equal(t, c.expectedSOTW, store.StateOfWorld())

			// Processing an empty snapshot should be ignored and not change the state of the world
			store.ProcessSnapshot(ctx, &snapshotTypes.Snapshot{})
			assert.Equal(t, c.expectedSOTW, store.StateOfWorld())
		})
	}
}

func NewMockAPIDocsHTTPClient(t *testing.T, expectedRequestURL string, expectedRequestHost string, expectedRequestHeaders []agent.Header, content string, err error) agent.APIDocsHTTPClient {
	return &mockAPIDocsHTTPClient{
		t:                      t,
		expectedRequestURL:     expectedRequestURL,
		expectedRequestHost:    expectedRequestHost,
		expectedRequestHeaders: expectedRequestHeaders,
		resultContent:          content,
		resultErr:              err,
	}
}

type mockAPIDocsHTTPClient struct {
	t *testing.T

	expectedRequestURL     string
	expectedRequestHost    string
	expectedRequestHeaders []agent.Header

	resultContent string
	resultErr     error
}

func (c *mockAPIDocsHTTPClient) Get(ctx context.Context, requestURL *url.URL, requestHost string, requestHeaders []agent.Header) ([]byte, error) {
	if c.expectedRequestURL == "" {
		c.t.Errorf("unexpected call to APIDocsHTTPClient.Get")
		c.t.Fail()
	}
	assert.Equal(c.t, c.expectedRequestURL, requestURL.String())
	assert.Equal(c.t, c.expectedRequestHost, requestHost)
	assert.Equal(c.t, c.expectedRequestHeaders, requestHeaders)

	return []byte(c.resultContent), c.resultErr
}
