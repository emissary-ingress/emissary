package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"
)

const (
	// DefDocsPrefix is the default prefix for documentation
	DefDocsPrefix = "/.ambassador-internal/"
)

type Store interface {
	AddServiceDocs(serviceDocs ServiceDocs, openAPIDoc *OpenAPIDoc) error
	DeleteServiceDocs(serviceDocs ServiceDocs) error
	GetAllServicesDocs() ServicesDocsMap
}

type APIDocsStore struct {
	// TODO(alexgervais): Document all the fields
	snapshot        *CoreSnapshot
	store           Store
	client          *APIDocsHTTPClient
	serviceDocsDiff *serviceDocsDiffCalculator
}

// NewAPIDocsStore is the main APIDocsStore constructor.
func NewAPIDocsStore(ctx context.Context) *APIDocsStore {
	dlog.Debug(ctx, "Creating new APIDocsStore")
	return &APIDocsStore{
		client:          newAPIDocsHTTPClient(ctx),
		store:           newInMemoryStore(),
		serviceDocsDiff: newServiceDocsCalculator([]ServiceDocs{}),
	}
}

func (a *APIDocsStore) ProcessSnapshot(ctx context.Context, snapshot *snapshotTypes.Snapshot) {
	dlog.Debug(ctx, "Processing snapshot...")
	a.retrieve(ctx, snapshot)
}

// StateOfWorld returns the current state of all discovered API docs.
func (a *APIDocsStore) StateOfWorld() []*snapshotTypes.APIDoc {
	dlog.Debug(context.Background(), "Building and returning StateOfWorld...")
	return toAPIDocs(a.store.GetAllServicesDocs())
}

func (a *APIDocsStore) retrieve(ctx context.Context, snapshot *snapshotTypes.Snapshot) {
	// TODO(alexgervais): Validate the filter protection in Emissary-ingress
	// TODO(alexgervais): Do we still want the "/.ambassador-internal/" default value? It's been deprecated.
	// TODO(alexgervais): Filter by namespaces or labels? Skip internal mappings?

	defer func() {
		// Once we are finished retrieving service docs, delete anything we
		// don't need anymore
		a.serviceDocsDiff.DeleteOld(ctx, a.store)

		dlog.Debug(ctx, "Iteration done")
	}()

	// Ignoring DevPortal configurations for now...

	dlog.Debugf(ctx, "Found %d Mappings", len(snapshot.Kubernetes.Mappings))
	for _, mapping := range snapshot.Kubernetes.Mappings {
		if mapping == nil {
			continue
		}
		mappingDocs := mapping.Spec.Docs
		if mappingDocs == nil || (mappingDocs.Ignored != nil && *mappingDocs.Ignored == true) {
			continue
		}
		displayName := mappingDocs.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("%s.%s", mapping.GetName(), mapping.GetNamespace())
		}
		mappingHeaders := a.getMappingHeaders(mapping.Spec.Headers)
		mappingPrefix := mapping.Spec.Prefix
		mappingRewrite := "/"
		if mapping.Spec.Rewrite != nil {
			mappingRewrite = *mapping.Spec.Rewrite
		}
		mappingHost := mapping.Spec.Host

		var openAPIDocs *OpenAPIDoc
		if mappingDocs.URL != "" {
			parsedURL, err := url.Parse(mappingDocs.URL)
			if err != nil {
				dlog.Errorf(ctx, "Could not parse URL or path in 'docs' %q", mappingDocs.URL)
				continue
			}
			dlog.Debugf(ctx, "'url' specified: querying %s", parsedURL)
			openAPIDocs = a.getDocs(ctx, parsedURL, "", "", mappingHeaders, mappingHost, false)
		} else {
			mappingDocsPath := mappingDocs.Path
			if mappingDocsPath != "" {
				// note: filepath.Join() does not work, because it removes any trailing `/`
				mappingDocsPath = strings.ReplaceAll(mappingRewrite+mappingDocsPath, "//", "/")
				dlog.Debugf(ctx, "'path' specified: resulting in %s", mappingDocsPath)
			}
			// TODO: Improve the way mappingsDocsURL is built, according to namespaces, ports and whatnot
			mappingsDocsURL, err := url.Parse(mapping.Spec.Service + mappingDocsPath)
			if err != nil {
				dlog.Errorf(ctx, "Could not parse URL or path in 'docs' %q", mappingsDocsURL)
				continue
			}

			mappingsDocsURL.Scheme = "http"
			dlog.Debugf(ctx, "'url' specified: querying %s", mappingsDocsURL)
			openAPIDocs = a.getDocs(ctx, mappingsDocsURL, mappingHost, mappingPrefix, mappingHeaders, mappingHost, true)
		}

		if openAPIDocs != nil {
			dlog.Debugf(ctx, "Adding service docs")
			serviceDocs := ServiceDocs{
				Ref: &kates.ObjectReference{
					Kind:            mapping.Kind,
					Namespace:       mapping.Namespace,
					Name:            mapping.Name,
					UID:             mapping.UID,
					APIVersion:      mapping.APIVersion,
					ResourceVersion: mapping.ResourceVersion,
				},
				Name: displayName,
			}
			err := a.store.AddServiceDocs(serviceDocs, openAPIDocs)
			if err == nil {
				a.serviceDocsDiff.Add(serviceDocs)
			}
		}
	}
}

func (a *APIDocsStore) getDocs(ctx context.Context, u *url.URL, host string, prefix string, mappingHeaders []MappingHeader, publicHost string, keep bool) *OpenAPIDoc {
	dlog.Debugf(ctx, "GET start %s", u)
	b, err := a.client.Get(u, host, mappingHeaders)
	if err != nil {
		dlog.Errorf(ctx, "GET failed %s", u)
		return nil
	}

	var d *OpenAPIDoc
	dlog.Debugf(ctx, "GET done %s", u)
	if b != nil {
		d = NewOpenAPI(ctx, b, publicHost, prefix, keep)
	}
	return d
}

type OpenAPIDoc struct {
	JSON    []byte
	Type    string
	Version string
}

func NewOpenAPI(ctx context.Context, docBytes []byte, baseURL string, prefix string, keep bool) *OpenAPIDoc {
	dlog.Debugf(ctx, "Trying to create new OpenAPI doc: base_url=%q prefix=%q", baseURL, prefix)

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(docBytes)
	if err != nil {
		dlog.Errorln(ctx, "Failed to load OpenAPI spec:", err)
		return nil
	}
	err = doc.Validate(loader.Context)
	if err != nil {
		dlog.Errorln(ctx, "Failed to validate OpenAPI spec:", err)
		return nil
	}

	// Get prefix out of first server URL. E.g. if it's
	// http://example.com/v1, we want to to add /v1 after the Ambassador
	// prefix.
	existingPrefix := ""
	if doc.Servers != nil && doc.Servers[0] != nil {
		currentServerURL := doc.Servers[0].URL
		dlog.Debugf(ctx, "Checking first server's URL: url=%#v", currentServerURL)
		existingUrl, err := url.Parse(currentServerURL)
		if err == nil {
			existingPrefix = existingUrl.Path
		} else {
			dlog.Errorf(ctx, "Failed to parse 'servers' URL: url=%q: %v", currentServerURL, err)
		}
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		dlog.Debugf(ctx, "Could not parse URL %q", baseURL)
	} else {
		if prefix != "" {
			if existingPrefix != "" && keep {
				base.Path = path.Join(base.Path, prefix, existingPrefix)
			} else {
				base.Path = path.Join(base.Path, prefix)
			}
		} else {
			base.Path = existingPrefix
		}

		doc.Servers = []*openapi3.Server{{
			URL: base.String(),
		}}
	}

	json, err := doc.MarshalJSON()
	if err != nil {
		dlog.Errorln(ctx, "Failed to marshal OpenAPI spec:", err)
		return nil
	}

	return &OpenAPIDoc{
		JSON:    json,
		Type:    "OpenAPI",
		Version: "v3",
	}
}

func (a *APIDocsStore) getMappingHeaders(headers map[string]amb.BoolOrString) []MappingHeader {
	mappingHeaders := []MappingHeader{}

	for key, headerValue := range headers {
		if key == ":authority" {
			continue
		}
		value := ""
		if headerValue.String != nil {
			value = *headerValue.String
		}
		if headerValue.Bool != nil {
			value = strconv.FormatBool(*headerValue.Bool)
		}
		mappingHeaders = append(mappingHeaders, MappingHeader{name: key, value: value})
	}

	return mappingHeaders
}

type MappingHeader struct {
	name  string
	value string
}

type APIDocsHTTPClient struct {
	client *http.Client
	ctx    context.Context
}

func newAPIDocsHTTPClient(ctx context.Context) *APIDocsHTTPClient {
	dialer := &net.Dialer{
		Timeout: time.Second * 2,
	}
	c := &http.Client{
		Timeout: time.Second * 2,

		// TODO: We should make this an explicit opt-in
		Transport: &http.Transport{
			/* #nosec */
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Dial:            dialer.Dial,
		},
	}
	ctx = dlog.WithField(ctx, "component", "ambassador-agent")
	return &APIDocsHTTPClient{
		client: c,
		ctx:    ctx,
	}
}

func (c *APIDocsHTTPClient) Get(requestURL *url.URL, mHost string, mHeaders []MappingHeader) ([]byte, error) {
	ctx := dlog.WithField(c.ctx, "url", requestURL)
	ctx = dlog.WithField(ctx, "mhost", mHost)

	dlog.Debugf(ctx, "HTTP GET URL: %s Host: %s Headers: %v\n", requestURL.String(), mHost, mHeaders)
	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		dlog.Debugf(ctx, "http.NewRequest for %s failed", requestURL.String())
		dlog.Error(ctx, err)
		return nil, err
	}
	req.Close = true

	if mHost != "" {
		dlog.Debugf(ctx, "Using host=%s", mHost)
		req.Host = mHost
	}

	if mHeaders != nil {
		for _, header := range mHeaders {
			dlog.Debugf(ctx, "Adding header %s=%s", header.name, header.value)
			req.Header.Set(header.name, header.value)
		}
	}

	res, err := c.client.Do(req)
	if err != nil {
		dlog.Debugf(ctx, "client.Do for %s failed", requestURL.String())
		dlog.Error(ctx, err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		dlog.Errorf(ctx, "Bad HTTP request: status_code=%v", res.StatusCode)
		return nil, fmt.Errorf("HTTP error %d from %s", res.StatusCode, requestURL)
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read HTTP response body")
	}

	dlog.Debug(ctx, "GET succeeded")
	return buf, nil
}

type ServiceDocs struct {
	Ref  *kates.ObjectReference
	Name string
}

type serviceMap map[ServiceDocs]bool

// Figure out what services no longer exist and need to be deleted.
type serviceDocsDiffCalculator struct {
	previous serviceMap
	current  serviceMap
}

// newServiceDocsCalculator creates a new diff calculator for service docs
func newServiceDocsCalculator(known []ServiceDocs) *serviceDocsDiffCalculator {
	knownMap := make(serviceMap)
	for _, service := range known {
		knownMap[service] = true
	}
	return &serviceDocsDiffCalculator{current: make(serviceMap), previous: knownMap}
}

// Done retrieving all known services: this will return list of services to
// delete.
func (d *serviceDocsDiffCalculator) NewRound() []ServiceDocs {
	toDelete := make([]ServiceDocs, 0)
	for service := range d.previous {
		if !d.current[service] {
			toDelete = append(toDelete, service)
		}
	}
	d.previous = d.current
	d.current = make(serviceMap)
	return toDelete
}

// Add a ServiceDocs that was successfully retrieved this round
func (d *serviceDocsDiffCalculator) Add(s ServiceDocs) {
	d.current[s] = true
}

// DeleteOld deletes old service docs that are no longer present
func (d *serviceDocsDiffCalculator) DeleteOld(ctx context.Context, store Store) {
	// Finished retrieving services, so delete any we don't recognize:
	for _, svc := range d.NewRound() {
		dlog.Debugf(ctx, "Deleting old Service Docs %s", svc)
		store.DeleteServiceDocs(svc)
	}
}

type ServicesDocsMap map[ServiceDocs]*OpenAPIDoc

type inMemoryStore struct {
	metadatamutex sync.RWMutex
	metadata      ServicesDocsMap
}

func newInMemoryStore() *inMemoryStore {
	res := &inMemoryStore{
		metadata: make(ServicesDocsMap),
	}

	return res
}

func (s *inMemoryStore) AddServiceDocs(ks ServiceDocs, openAPIDoc *OpenAPIDoc) error {
	s.metadatamutex.Lock()
	defer s.metadatamutex.Unlock()

	s.metadata[ks] = openAPIDoc
	return nil
}

func (s *inMemoryStore) DeleteServiceDocs(ks ServiceDocs) error {
	s.metadatamutex.Lock()
	defer s.metadatamutex.Unlock()

	delete(s.metadata, ks)
	return nil
}

func (s *inMemoryStore) GetAllServicesDocs() ServicesDocsMap {
	s.metadatamutex.RLock()
	defer s.metadatamutex.RUnlock()
	return s.metadata
}

func toAPIDocs(serviceDocsMap ServicesDocsMap) []*snapshotTypes.APIDoc {
	results := make([]*snapshotTypes.APIDoc, 0)
	for serviceDocs, openAPIDocs := range serviceDocsMap {
		if openAPIDocs != nil {
			dlog.Debugf(context.Background(), "Reporting Mapping %v docs", serviceDocs.Ref)
			apiDoc := &snapshotTypes.APIDoc{
				Data: openAPIDocs.JSON,
				TypeMeta: &kates.TypeMeta{
					Kind:       openAPIDocs.Type,
					APIVersion: openAPIDocs.Version,
				},
				Metadata: &kates.ObjectMeta{
					Name: serviceDocs.Name,
				},
				TargetRef: serviceDocs.Ref,
			}
			results = append(results, apiDoc)
		}
	}
	return results
}
