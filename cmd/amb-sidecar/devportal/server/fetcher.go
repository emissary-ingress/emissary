package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	log "github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/amb-sidecar/internalaccess"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
)

// Retrieve a URL.
type HTTPGetFunc func(requestURL *url.URL, internalSecret string, logger *log.Entry) ([]byte, error)

type serviceMap map[Service]bool

// Figure out what services no longer exist and need to be deleted.
type diffCalculator struct {
	previous serviceMap
	current  serviceMap
}

func NewDiffCalculator(known []Service) *diffCalculator {
	knownMap := make(serviceMap)
	for _, service := range known {
		knownMap[service] = true
	}
	return &diffCalculator{current: make(serviceMap), previous: knownMap}
}

// Done retrieving all known services: this will return list of services to
// delete.
func (d *diffCalculator) NewRound() []Service {
	toDelete := make([]Service, 0)
	for service := range d.previous {
		if !d.current[service] {
			toDelete = append(toDelete, service)
		}
	}
	d.previous = d.current
	d.current = make(serviceMap)
	return toDelete
}

// Add a Service that was successfully retrieved this round
func (d *diffCalculator) Add(s Service) {
	d.current[s] = true
}

type MappingObserverFunc func(prefix, rewrite string) bool

type observer struct {
	observer   MappingObserverFunc
	lastPrefix string
	ret        bool
}

type fetcher struct {
	store     ServiceStore
	httpGet   HTTPGetFunc
	retriever chan chan bool
	diff      *diffCalculator

	logger *log.Entry
	cfg    types.Config

	// Shared secret to send so that we can access .ambassador-internal
	internalSecret *internalaccess.InternalSecret

	observers map[string]*observer
}

type MappingSubscriptions interface {
	SubscribeMappingObserver(mappingName string, observe MappingObserverFunc)
}

type ServiceStore interface {
	Init(fetcher MappingSubscriptions)
	AddService(service Service, baseURL string, prefix string, openAPIDoc []byte) error
	DeleteService(service Service) error
}

// Object that retrieves service info and OpenAPI docs (if available) and
// adds/deletes changes from last run.
func NewFetcher(
	store ServiceStore,
	httpGet HTTPGetFunc,
	known []Service,
	cfg types.Config,
) *fetcher {
	f := &fetcher{
		store:          store,
		httpGet:        httpGet,
		retriever:      make(chan chan bool),
		diff:           NewDiffCalculator(known),
		logger:         log.WithFields(log.Fields{"subsystem": "fetcher"}),
		cfg:            cfg,
		internalSecret: internalaccess.GetInternalSecret(),
		observers:      make(map[string]*observer),
	}
	store.Init(f)
	return f
}

func (f *fetcher) Run(ctx context.Context) {
	f._retrieve("request")
	ticker := time.NewTicker(f.cfg.DevPortalPollInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			f._retrieve("timer")
		case ack := <-f.retriever:
			f._retrieve("request")
			ack <- true
		}
	}
}

// Get a string attribute of a JSON object:
func getString(o *gabs.Container, attr string) string {
	return o.Path(attr).Data().(string)
}

var dialer = &net.Dialer{
	Timeout: time.Second * 2,
}

var client = util.SimpleClient{Client: &http.Client{
	Timeout: time.Second * 2,

	// TODO: We should make this an explicit opt-in
	Transport: &http.Transport{
		/* #nosec */
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial:            dialer.Dial,
	},
}}

func HTTPGet(requestURL *url.URL, internalSecret string, logger *log.Entry) ([]byte, error) {
	logger = logger.WithFields(log.Fields{"url": requestURL})
	logger.Debug("HTTP GET")
	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	req.Header.Set("X-Ambassador-Internal-Auth", internalSecret)
	req.Close = true

	buf, err := client.DoBodyBytes(req, func(response *http.Response, body []byte) (err error) {
		if response.StatusCode != 200 {
			logger.WithFields(
				log.Fields{"status_code": response.StatusCode}).Error(
				"Bad HTTP response")
			err = fmt.Errorf("HTTP error %d from %s", response.StatusCode, requestURL)
		}
		return
	})
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	logger.Debug("GET succeeded")
	return buf, nil
}

func (f *fetcher) Retrieve() {
	waiter := make(chan bool)
	f.retriever <- waiter
	<-waiter
}

func (f *fetcher) _retrieve(reason string) {
	f.logger.Debug("Iteration started ", reason, " ")
	requestURL, err := f.cfg.AmbassadorAdminURL.Parse("/ambassador/v0/diag/?json=true")
	if err != nil {
		// This should _never_ happen; cfg has alread been
		// validated, and the string is fixex.
		panic(err)
	}
	buf, err := f.httpGet(requestURL, "", f.logger)
	if err != nil {
		log.Print(err)
		return
	}
	// Don't bother looking at error; failed queries will result in service
	// being removed from Dev Portal.

	json, err := gabs.ParseJSON(buf)
	if err != nil {
		log.Print(err)
		return
	}
	children, err := json.S("groups").ChildrenMap()
	if err != nil {
		log.Print(err)
		return
	}
	for _, child := range children {
		// We don't consider inactive services:
		if !child.S("_active").Data().(bool) {
			continue
		}
		// We don't consider non-HTTP services:
		if getString(child, "kind") != "IRHTTPMappingGroup" {
			continue
		}
		mappings, err := child.S("mappings").Children()
		if err != nil {
			f.logger.WithError(err).Error("No mappings JSON entry")
			return
		}
		for _, mapping := range mappings {
			location := getString(mapping, "location")
			if location == "--internal--" {
				continue
			}
			location_parts := strings.Split(location, ".")
			if len(location_parts) < 2 {
				// This is most likely a Knative mapping: ignore for now.
				// See apro issue #618 https://github.com/datawire/apro/issues/618
				continue
			}
			prefix := getString(mapping, "prefix")
			rewrite := getString(mapping, "rewrite")
			clusterName := getString(mapping, "cluster.name")
			prefix = strings.TrimRight(prefix, "/")
			name := location_parts[0]
			namespace := location_parts[1]
			var baseURL string
			if mapping.Exists("host") {
				// TODO what if it's http? (arguably it should never be)
				baseURL = "https://" + getString(mapping, "host")
			} else {
				baseURL = f.cfg.AmbassadorExternalURL.String()
			}
			mappingName := getString(mapping, "name")
			if f.observeInternalMapping(mappingName, prefix, rewrite) || f.isInternalCluster(clusterName) {
				f.logger.WithFields(log.Fields{
					"mappingName": mappingName,
					"name":        name,
					"namespace":   namespace,
					"baseURL":     baseURL,
					"prefix":      prefix,
					"rewrite":     rewrite,
					"clusterName": clusterName,
				}).Debug("Found internal mapping, skipping")
				continue
			}
			f.logger.WithFields(log.Fields{
				"name":      name,
				"namespace": namespace,
				"baseURL":   baseURL,
				"prefix":    prefix,
			}).Debug("Found mapping")
			// Get the OpenAPI documentation:
			var doc []byte
			requestURL, err := f.cfg.AmbassadorInternalURL.Parse(prefix + "/.ambassador-internal/openapi-docs")
			if err == nil {
				docBuf, err := f.httpGet(
					requestURL,
					f.internalSecret.Get(),
					f.logger)
				if err == nil {
					doc = docBuf
				}
			}
			_, err = gabs.ParseJSON(doc)
			if err != nil {
				doc = nil
			}
			service := Service{Namespace: namespace, Name: name}
			err = f.store.AddService(service, baseURL, prefix, doc)
			if err == nil {
				f.diff.Add(service)
			}
		}
	}

	// Finished retrieving services, so delete any we don't recognize:
	for _, service := range f.diff.NewRound() {
		f.logger.WithFields(log.Fields{
			"name": service.Name, "namespace": service.Namespace,
		}).Debug("Deleting old service we didn't find in this iteration")
		f.store.DeleteService(service)
	}
	f.logger.Debug("Iteration done")
}

func (f *fetcher) observeInternalMapping(mappingName, prefix, rewrite string) bool {
	observer, ok := f.observers[mappingName]
	if !ok {
		return false
	}
	return observer.observe(prefix, rewrite)
}

func (f *fetcher) SubscribeMappingObserver(mappingName string, observe MappingObserverFunc) {
	f.observers[mappingName] = &observer{
		observer: observe,
	}
}

func (o *observer) observe(prefix, rewrite string) bool {
	if o.lastPrefix != prefix {
		o.ret = o.observer(prefix, rewrite)
		o.lastPrefix = prefix
	}
	return o.ret
}

func (f *fetcher) isInternalCluster(clusterName string) bool {
	if strings.HasPrefix(clusterName, "cluster_127_0_0_1") {
		return true
	}
	return false
}
