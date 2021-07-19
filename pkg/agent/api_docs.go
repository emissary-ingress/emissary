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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// APIDocsStore is responsible for collecting the API docs from Mapping resources in a k8s cluster.
type APIDocsStore struct {
	// Client is used to scrape all Mappings for API documentation
	Client APIDocsHTTPClient
	// store hold the state of the world, with all Mappings and their API docs
	store *inMemoryStore
	// docsDiff helps calculate whether an API doc should be kept or discarded after processing a snapshot
	docsDiff *docsDiffCalculator
}

// NewAPIDocsStore is the main APIDocsStore constructor.
func NewAPIDocsStore() *APIDocsStore {
	return &APIDocsStore{
		Client:   newAPIDocsHTTPClient(),
		store:    newInMemoryStore(),
		docsDiff: newMappingDocsCalculator([]mappingDoc{}),
	}
}

// ProcessSnapshot will query the required services to retrieve the API documentation for each
// of the Mappings in the snapshot
func (a *APIDocsStore) ProcessSnapshot(ctx context.Context, snapshot *snapshotTypes.Snapshot) {
	dlog.Debug(ctx, "Processing snapshot...")
	a.retrieve(ctx, snapshot)
}

// StateOfWorld returns the current state of all discovered API docs.
func (a *APIDocsStore) StateOfWorld() []*snapshotTypes.APIDoc {
	return toAPIDocs(a.store.getAllMappingDocs())
}

func (a *APIDocsStore) retrieve(ctx context.Context, snapshot *snapshotTypes.Snapshot) {
	defer func() {
		// Once we are finished retrieving mapping docs, delete anything we
		// don't need anymore
		a.docsDiff.deleteOld(ctx, a.store)

		dlog.Debug(ctx, "Iteration done")
	}()

	if snapshot == nil || snapshot.Kubernetes == nil {
		return
	}

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
		mappingHeaders := a.buildMappingRequestHeaders(mapping.Spec.Headers)
		mappingPrefix := mapping.Spec.Prefix
		mappingRewrite := "/"
		if mapping.Spec.Rewrite != nil {
			mappingRewrite = *mapping.Spec.Rewrite
		}
		mappingHost := mapping.Spec.Host

		var doc *openAPIDoc
		if mappingDocs.URL != "" {
			parsedURL, err := url.Parse(mappingDocs.URL)
			if err != nil {
				dlog.Errorf(ctx, "could not parse URL or path in 'docs' %q", mappingDocs.URL)
				continue
			}

			dlog.Debugf(ctx, "'url' specified: querying %s", parsedURL)
			doc = a.getDoc(ctx, parsedURL, "", mappingHeaders, mappingHost, "", false)
		} else {
			mappingDocsPath := mappingDocs.Path
			if mappingDocsPath != "" {
				// note: filepath.Join() does not work, because it removes any trailing `/`
				mappingDocsPath = strings.ReplaceAll(mappingRewrite+mappingDocsPath, "//", "/")
				dlog.Debugf(ctx, "'path' specified: resulting in %s", mappingDocsPath)
			}

			// TODO(alexgervais): Improve the way mappingsDocsURL is built, according to namespaces, ports and whatnot
			mappingsDocsURL, err := url.Parse(mapping.Spec.Service + mappingDocsPath)
			if err != nil {
				dlog.Errorf(ctx, "could not parse URL or path in 'docs' %q", mappingsDocsURL)
				continue
			}
			mappingsDocsURL.Scheme = "http"

			dlog.Debugf(ctx, "'url' specified: querying %s", mappingsDocsURL)
			doc = a.getDoc(ctx, mappingsDocsURL, mappingHost, mappingHeaders, mappingHost, mappingPrefix, true)
		}

		if doc != nil {
			dlog.Debugf(ctx, "Adding mapping docs")
			md := mappingDoc{
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
			err := a.store.add(md, doc)
			if err == nil {
				a.docsDiff.add(md)
			}
		}
	}
}

func (a *APIDocsStore) getDoc(ctx context.Context, queryURL *url.URL, queryHost string, queryHeaders []Header, publicHost string, prefix string, keep bool) *openAPIDoc {
	b, err := a.Client.Get(ctx, queryURL, queryHost, queryHeaders)
	if err != nil {
		dlog.Errorf(ctx, "get failed %s: %v", queryURL, err)
		return nil
	}

	if b != nil {
		return newOpenAPI(ctx, b, publicHost, prefix, keep)
	}
	return nil
}

// openAPIDoc represent a typed OpenAPI/Swagger document
type openAPIDoc struct {
	// The actual OpenAPI/Swagger document in JSON
	JSON []byte
	// The document type (OpenAPI)
	Type string
	// The document version (v3)
	Version string
}

// openAPIDoc constructor from raw bytes.
// The baseURL and prefix are used to edit the original document with server information to query the API publicly
func newOpenAPI(ctx context.Context, docBytes []byte, baseURL string, prefix string, keep bool) *openAPIDoc {
	dlog.Debugf(ctx, "Trying to create new OpenAPI doc: base_url=%q prefix=%q", baseURL, prefix)

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(docBytes)
	if err != nil {
		dlog.Errorln(ctx, "failed to load OpenAPI spec:", err)
		return nil
	}
	err = doc.Validate(loader.Context)
	if err != nil {
		dlog.Errorln(ctx, "failed to validate OpenAPI spec:", err)
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
			dlog.Errorf(ctx, "failed to parse 'servers' URL: url=%q: %v", currentServerURL, err)
		}
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		dlog.Debugf(ctx, "could not parse URL %q", baseURL)
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
		dlog.Errorln(ctx, "failed to marshal OpenAPI spec:", err)
		return nil
	}

	return &openAPIDoc{
		JSON:    json,
		Type:    "OpenAPI",
		Version: "v3",
	}
}

func (a *APIDocsStore) buildMappingRequestHeaders(mappingHeaders map[string]amb.BoolOrString) []Header {
	headers := []Header{}

	for key, headerValue := range mappingHeaders {
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
		headers = append(headers, Header{Name: key, Value: value})
	}

	return headers
}

type Header struct {
	Name  string
	Value string
}

type APIDocsHTTPClient interface {
	Get(ctx context.Context, requestURL *url.URL, requestHost string, requestHeaders []Header) ([]byte, error)
}

type apiDocsHTTPClient struct {
	*http.Client
}

func newAPIDocsHTTPClient() *apiDocsHTTPClient {
	dialer := &net.Dialer{
		Timeout: time.Second * 10,
	}
	c := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			/* #nosec */
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Dial:            dialer.Dial,
		},
	}
	return &apiDocsHTTPClient{c}
}

func (c *apiDocsHTTPClient) Get(ctx context.Context, requestURL *url.URL, requestHost string, requestHeaders []Header) ([]byte, error) {
	ctx = dlog.WithField(ctx, "url", requestURL)
	ctx = dlog.WithField(ctx, "host", requestHost)

	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		dlog.Error(ctx, err)
		return nil, err
	}
	req.Close = true

	if requestHost != "" {
		dlog.Debugf(ctx, "Using host=%s", requestHost)
		req.Host = requestHost
	}

	if requestHeaders != nil {
		for _, queryHeader := range requestHeaders {
			dlog.Debugf(ctx, "Adding header %s=%s", queryHeader.Name, queryHeader.Value)
			req.Header.Set(queryHeader.Name, queryHeader.Value)
		}
	}

	res, err := c.Do(req)
	if err != nil {
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

	return buf, nil
}

// mappingDoc holds a reference to a Mapping with a 'docs' attribute, for a given display name.
type mappingDoc struct {
	Ref  *kates.ObjectReference
	Name string
}

type mappingDocMap map[mappingDoc]bool

// Figure out which Mapping and API doc no longer exist and need to be deleted.
type docsDiffCalculator struct {
	previous mappingDocMap
	current  mappingDocMap
}

// newMappingDocsCalculator creates a new diff calculator for mapping docs
func newMappingDocsCalculator(known []mappingDoc) *docsDiffCalculator {
	knownMap := make(mappingDocMap)
	for _, m := range known {
		knownMap[m] = true
	}
	return &docsDiffCalculator{current: make(mappingDocMap), previous: knownMap}
}

// After retrieving all known mappings, newRound will return list of mapping docs to delete
func (d *docsDiffCalculator) newRound() []mappingDoc {
	toDelete := make([]mappingDoc, 0)
	for md := range d.previous {
		if !d.current[md] {
			toDelete = append(toDelete, md)
		}
	}
	d.previous = d.current
	d.current = make(mappingDocMap)
	return toDelete
}

// add a MappingDoc that was successfully retrieved this round
func (d *docsDiffCalculator) add(md mappingDoc) {
	d.current[md] = true
}

// deleteOld deletes old MappingDocs that are no longer present
func (d *docsDiffCalculator) deleteOld(ctx context.Context, store *inMemoryStore) {
	for _, md := range d.newRound() {
		dlog.Debugf(ctx, "Deleting old Mapping Docs %s", md)
		store.delete(md)
	}
}

type mappingDocsMap map[mappingDoc]*openAPIDoc

type inMemoryStore struct {
	entriesMutex sync.RWMutex
	entries      mappingDocsMap
}

func newInMemoryStore() *inMemoryStore {
	res := &inMemoryStore{
		entries: make(mappingDocsMap),
	}

	return res
}

func (s *inMemoryStore) add(md mappingDoc, openAPIDoc *openAPIDoc) error {
	s.entriesMutex.Lock()
	defer s.entriesMutex.Unlock()

	s.entries[md] = openAPIDoc
	return nil
}

func (s *inMemoryStore) delete(md mappingDoc) error {
	s.entriesMutex.Lock()
	defer s.entriesMutex.Unlock()

	delete(s.entries, md)
	return nil
}

func (s *inMemoryStore) getAllMappingDocs() mappingDocsMap {
	s.entriesMutex.RLock()
	defer s.entriesMutex.RUnlock()

	return s.entries
}

func toAPIDocs(mappingDocsMap mappingDocsMap) []*snapshotTypes.APIDoc {
	results := make([]*snapshotTypes.APIDoc, 0)
	for md, openAPIDocs := range mappingDocsMap {
		if openAPIDocs != nil {
			apiDoc := &snapshotTypes.APIDoc{
				Data: openAPIDocs.JSON,
				TypeMeta: &kates.TypeMeta{
					Kind:       openAPIDocs.Type,
					APIVersion: openAPIDocs.Version,
				},
				Metadata: &kates.ObjectMeta{
					Name: md.Name,
				},
				TargetRef: md.Ref,
			}
			results = append(results, apiDoc)
		}
	}
	return results
}
