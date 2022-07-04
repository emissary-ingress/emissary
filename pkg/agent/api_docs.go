package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/datawire/dlib/dlog"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"
)

// APIDocsStore is responsible for collecting the API docs from Mapping resources in a k8s cluster.
type APIDocsStore struct {
	// Client is used to scrape all Mappings for API documentation
	Client APIDocsHTTPClient
	// DontProcessSnapshotBeforeTime keeps track of the moment the next received snapshot should be processed
	DontProcessSnapshotBeforeTime time.Time

	// store hold the state of the world, with all Mappings and their API docs
	store *inMemoryStore
	// docsDiff helps calculate whether an API doc should be kept or discarded after processing a snapshot
	docsDiff *docsDiffCalculator
	// processingSnapshotMutex holds a lock so that a single snapshot gets processed at a time
	processingSnapshotMutex sync.RWMutex
}

// NewAPIDocsStore is the main APIDocsStore constructor.
func NewAPIDocsStore() *APIDocsStore {
	return &APIDocsStore{
		Client:                        newAPIDocsHTTPClient(),
		DontProcessSnapshotBeforeTime: time.Unix(0, 0),

		store:    newInMemoryStore(),
		docsDiff: newMappingDocsCalculator([]docMappingRef{}),
	}
}

// ProcessSnapshot will query the required services to retrieve the API documentation for each
// of the Mappings in the snapshot. It will execute at most once every minute.
func (a *APIDocsStore) ProcessSnapshot(ctx context.Context, snapshot *snapshotTypes.Snapshot) {
	a.processingSnapshotMutex.Lock()
	defer a.processingSnapshotMutex.Unlock()

	emptyStore := len(a.store.getAll()) == 0
	mappings := getProcessableMappingsFromSnapshot(snapshot)
	if len(mappings) == 0 && emptyStore {
		dlog.Debug(ctx, "Skipping apidocs snapshot processing until a mapping with documentation is found")
		return
	}

	now := time.Now()
	if now.Before(a.DontProcessSnapshotBeforeTime) {
		dlog.Debugf(ctx, "Skipping apidocs snapshot processing until %v", a.DontProcessSnapshotBeforeTime)
		return
	}

	dlog.Debug(ctx, "Processing snapshot...")
	a.DontProcessSnapshotBeforeTime = now.Add(1 * time.Minute)

	if emptyStore {
		// We don't have anything in memory...
		// Retrieve API docs synchronously so it appears snappy to the first-time user,
		// or when the agent starts.
		a.scrape(ctx, mappings)
	} else {
		// This is just an update, it can be processed asynchronously.
		go a.scrape(ctx, mappings)
	}
}

// StateOfWorld returns the current state of all discovered API docs.
func (a *APIDocsStore) StateOfWorld() []*snapshotTypes.APIDoc {
	return toAPIDocs(a.store.getAll())
}

func getProcessableMappingsFromSnapshot(snapshot *snapshotTypes.Snapshot) []*amb.Mapping {
	processableMappings := []*amb.Mapping{}
	if snapshot == nil || snapshot.Kubernetes == nil {
		return processableMappings
	}

	for _, mapping := range snapshot.Kubernetes.Mappings {
		if mapping == nil {
			continue
		}
		mappingDocs := mapping.Spec.Docs
		if mappingDocs == nil || (mappingDocs.Ignored != nil && *mappingDocs.Ignored == true) {
			continue
		}
		processableMappings = append(processableMappings, mapping)
	}
	return processableMappings
}

// scrape will take care of fetching OpenAPI documentation from each of the
// Mappings resources as we process a snapshot.
//
// Be careful as there is a very similar implementation of this logic in the DevPortal which
// uses the ambassador diag representation to retrieve OpenAPI documentation from
// Mapping resources.
// Since both the DevPortal and the agent make use of this `docs` property, evolutions
// made here should be considered for DevPortal too.
func (a *APIDocsStore) scrape(ctx context.Context, mappings []*amb.Mapping) {
	defer func() {
		// Once we are finished retrieving mapping docs, delete anything we
		// don't need anymore
		a.docsDiff.deleteOld(ctx, a.store)
		dlog.Debug(ctx, "Iteration done")
	}()

	dlog.Debugf(ctx, "Found %d Mappings", len(mappings))
	for _, mapping := range mappings {
		mappingDocs := mapping.Spec.Docs
		displayName := mappingDocs.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("%s.%s", mapping.GetName(), mapping.GetNamespace())
		}
		mappingHeaders := buildMappingRequestHeaders(mapping.Spec.Headers)
		mappingPrefix := mapping.Spec.Prefix
		// Lookup the Hostname first since it is more restrictive, otherwise fallback on the Host attribute
		mappingHostname := mapping.Spec.Hostname
		if mappingHostname == "" || mappingHostname == "*" {
			mappingHostname = mapping.Spec.DeprecatedHost
		}

		dm := &docMappingRef{
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
		a.docsDiff.add(ctx, dm)

		var doc *openAPIDoc
		if mappingDocs.URL != "" {
			parsedURL, err := url.Parse(mappingDocs.URL)
			if err != nil {
				dlog.Errorf(ctx, "could not parse URL or path in 'docs' %q", mappingDocs.URL)
				continue
			}
			dlog.Debugf(ctx, "'url' specified: querying %s", parsedURL)
			doc = a.getDoc(ctx, parsedURL, "", mappingHeaders, mappingHostname, "", false)
		} else {
			mappingsDocsURL, err := extractQueryableDocsURL(mapping)
			if err != nil {
				dlog.Errorf(ctx, "could not parse URL or path in 'docs': %v", err)
				continue
			}
			dlog.Debugf(ctx, "'url' specified: querying %s", mappingsDocsURL)
			doc = a.getDoc(ctx, mappingsDocsURL, mappingHostname, mappingHeaders, mappingHostname, mappingPrefix, true)
		}

		if doc != nil {
			a.store.add(dm, doc)
		}
	}
}

func extractQueryableDocsURL(mapping *amb.Mapping) (*url.URL, error) {
	mappingDocsPath := mapping.Spec.Docs.Path
	mappingRewrite := "/"
	if mapping.Spec.Rewrite != nil {
		mappingRewrite = *mapping.Spec.Rewrite
	}
	if mappingDocsPath != "" {
		mappingDocsPath = strings.ReplaceAll(mappingRewrite+mappingDocsPath, "//", "/")
	}

	mappingsDocsURL, err := url.Parse(mapping.Spec.Service + mappingDocsPath)
	if err != nil {
		return nil, err
	}
	if mappingsDocsURL.Host == "" {
		// We did our best to parse the service+path, but failed to actually extract a Host.
		// Now, be more explicit about which is which.
		mappingsDocsURL.Host = mapping.Spec.Service
		mappingsDocsURL.Path = mappingDocsPath
		mappingsDocsURL.Scheme = ""
		mappingsDocsURL.Opaque = ""
		mappingsDocsURL = mappingsDocsURL.ResolveReference(mappingsDocsURL)
	}
	if !strings.Contains(mappingsDocsURL.Hostname(), ".") {
		// The host does not appear to be a TLD, append the namespace
		servicePort := mappingsDocsURL.Port()
		mappingsDocsURL.Host = fmt.Sprintf("%s.%s", mappingsDocsURL.Hostname(), mapping.Namespace)
		if servicePort != "" {
			mappingsDocsURL.Host = fmt.Sprintf("%s:%s", mappingsDocsURL.Hostname(), servicePort)
		}
	}
	if mappingsDocsURL.Scheme == "" {
		// Assume plain-text if the mapping.Spec.Service did not specify https
		mappingsDocsURL.Scheme = "http"
	}

	return mappingsDocsURL, nil
}

func (a *APIDocsStore) getDoc(ctx context.Context, queryURL *url.URL, queryHost string, queryHeaders []Header, publicHost string, prefix string, keepExistingPrefix bool) *openAPIDoc {
	b, err := a.Client.Get(ctx, queryURL, queryHost, queryHeaders)
	if err != nil {
		dlog.Errorf(ctx, "get failed %s: %v", queryURL, err)
		return nil
	}

	if b != nil {
		return newOpenAPI(ctx, b, publicHost, prefix, keepExistingPrefix)
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
func newOpenAPI(ctx context.Context, docBytes []byte, baseURL string, prefix string, keepExistingPrefix bool) *openAPIDoc {
	dlog.Debugf(ctx, "Trying to create new OpenAPI doc: base_url=%q prefix=%q", baseURL, prefix)

	version, err := openAPIVersion(docBytes)

	if err != nil {
		dlog.Errorln(ctx, "failed to determine open api version:", err)
		return nil
	}

	if version == 2 {
		docBytes, err = convertToOpenAPIV3(docBytes)

		if err != nil {
			dlog.Errorln(ctx, "failed to convert open api v2 contract to v3:", err)
			return nil

		}
	}

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
			if existingPrefix != "" && keepExistingPrefix {
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

	j, err := doc.MarshalJSON()
	if err != nil {
		dlog.Errorln(ctx, "failed to marshal OpenAPI spec:", err)
		return nil
	}

	return &openAPIDoc{
		JSON:    j,
		Type:    "OpenAPI",
		Version: "v3",
	}

}

func openAPIVersion(docBytes []byte) (int, error) {
	var genericOpenAPI map[string]interface{}

	if err := json.Unmarshal(docBytes, &genericOpenAPI); err != nil {
		return -1, fmt.Errorf("failed to unmarshal open api spec: %w", err)
	}

	_, ok := genericOpenAPI["swagger"]

	if ok {
		return 2, nil
	}

	return 3, nil
}

func convertToOpenAPIV3(docBytes []byte) ([]byte, error) {
	v2Doc := &openapi2.T{}

	if err := json.Unmarshal(docBytes, v2Doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal v2 Openapi contract: %w", err)
	}

	v3Doc, err := openapi2conv.ToV3(v2Doc)

	if err != nil {
		return nil, fmt.Errorf("failed to convert v2 Openapi contract: %w", err)
	}

	docBytes, err = json.Marshal(v3Doc)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal v3 Openapi contract: %w", err)
	}

	return docBytes, err
}

func buildMappingRequestHeaders(mappingHeaders map[string]string) []Header {
	headers := []Header{}

	for key, value := range mappingHeaders {
		if key == ":authority" {
			continue
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

// docMappingRef holds a reference to a Mapping with a 'docs' attribute, for a given display name.
type docMappingRef struct {
	Ref  *kates.ObjectReference
	Name string
}

type mappingDocMap map[string]bool

// Figure out which Mapping and API doc no longer exist and need to be deleted.
type docsDiffCalculator struct {
	previous mappingDocMap
	current  mappingDocMap
}

// newMappingDocsCalculator creates a new diff calculator for mapping docs
func newMappingDocsCalculator(known []docMappingRef) *docsDiffCalculator {
	knownMap := make(mappingDocMap)
	for _, m := range known {
		knownMap[string(m.Ref.UID)] = true
	}
	return &docsDiffCalculator{current: make(mappingDocMap), previous: knownMap}
}

// After retrieving all known mappings, newRound will return list of mapping docs to delete
func (d *docsDiffCalculator) newRound() []string {
	mappingUIDsToDelete := make([]string, 0)

	for previousRef := range d.previous {
		if !d.current[previousRef] {
			mappingUIDsToDelete = append(mappingUIDsToDelete, string(previousRef))
		}
	}
	d.previous = d.current
	d.current = make(mappingDocMap)

	return mappingUIDsToDelete
}

// add a MappingDoc that was successfully retrieved this round
func (d *docsDiffCalculator) add(ctx context.Context, dm *docMappingRef) {
	if dm != nil && dm.Ref != nil {
		dlog.Debugf(ctx, "Adding Mapping Docs diff reference %s", dm)
		d.current[string(dm.Ref.UID)] = true
	}
}

// deleteOld deletes old MappingDocs that are no longer present
func (d *docsDiffCalculator) deleteOld(ctx context.Context, store *inMemoryStore) {
	for _, mappingUID := range d.newRound() {
		dlog.Debugf(ctx, "Deleting old Mapping Docs %s", mappingUID)
		store.deleteRefUID(mappingUID)
	}
}

type docsRef struct {
	docMappingRef *docMappingRef
	openAPIDoc    *openAPIDoc
}
type docsRefMap map[string]*docsRef

type inMemoryStore struct {
	entriesMutex sync.RWMutex
	entries      docsRefMap
}

func newInMemoryStore() *inMemoryStore {
	res := &inMemoryStore{
		entries: make(docsRefMap),
	}

	return res
}

func (s *inMemoryStore) add(dm *docMappingRef, openAPIDoc *openAPIDoc) {
	s.entriesMutex.Lock()
	defer s.entriesMutex.Unlock()

	s.entries[string(dm.Ref.UID)] = &docsRef{docMappingRef: dm, openAPIDoc: openAPIDoc}
}

func (s *inMemoryStore) deleteRefUID(mappingRefUID string) {
	s.entriesMutex.Lock()
	defer s.entriesMutex.Unlock()

	for entryUID := range s.entries {
		if mappingRefUID == entryUID {
			delete(s.entries, entryUID)
		}
	}
}

func (s *inMemoryStore) getAll() []*docsRef {
	s.entriesMutex.RLock()
	defer s.entriesMutex.RUnlock()

	var dr []*docsRef
	for _, e := range s.entries {
		dr = append(dr, e)
	}
	return dr
}

func toAPIDocs(docsRefs []*docsRef) []*snapshotTypes.APIDoc {
	results := make([]*snapshotTypes.APIDoc, 0)
	for _, doc := range docsRefs {
		if doc != nil && doc.docMappingRef != nil && doc.openAPIDoc != nil {
			apiDoc := &snapshotTypes.APIDoc{
				Data: doc.openAPIDoc.JSON,
				TypeMeta: &kates.TypeMeta{
					Kind:       doc.openAPIDoc.Type,
					APIVersion: doc.openAPIDoc.Version,
				},
				Metadata: &kates.ObjectMeta{
					Name: doc.docMappingRef.Name,
				},
				TargetRef: doc.docMappingRef.Ref,
			}
			results = append(results, apiDoc)
		}
	}
	return results
}
