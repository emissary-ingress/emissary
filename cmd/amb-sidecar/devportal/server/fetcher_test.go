package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"sort"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/kubernetes"
	"github.com/datawire/apro/cmd/amb-sidecar/devportal/openapi"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

var testdataAmbassadorDiagJSON, _ = ioutil.ReadFile("testdata/ambassador-diag.json")
var testdataOpenAPIDocsJSON, _ = ioutil.ReadFile("testdata/openapi-docs.json")

func TestDiffCalculator(t *testing.T) {
	g := NewGomegaWithT(t)
	A, B := kubernetes.Service{Name: "a"}, kubernetes.Service{Name: "b"}
	C, D := kubernetes.Service{Name: "c"}, kubernetes.Service{Name: "d"}

	// Starting point: we know about A and B
	calc := NewDiffCalculator([]kubernetes.Service{A, B})

	// Round 1: we detect A and C. That means B should be marked as deleted.
	calc.Add(A)
	calc.Add(C)
	g.Expect(calc.NewRound()).To(Equal([]kubernetes.Service{B}))

	// Round 2: we detect A and C. That means no deletes.
	calc.Add(A)
	calc.Add(C)
	g.Expect(calc.NewRound()).To(Equal([]kubernetes.Service{}))

	// Round 3: we detect A and C and D. That means no deletes.
	calc.Add(A)
	calc.Add(C)
	calc.Add(D)
	g.Expect(calc.NewRound()).To(Equal([]kubernetes.Service{}))

	// Round 4: we detect B and C. That means A and D are deleted.
	calc.Add(B)
	calc.Add(C)
	result := calc.NewRound()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	g.Expect(result).To(Equal([]kubernetes.Service{A, D}))

	// Round 5: we detect nothing. That means B and C are deleted.
	result = calc.NewRound()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	g.Expect(result).To(Equal([]kubernetes.Service{B, C}))
}

// Hard-code diagd output, as well as OpenAPI docs for one service:
func fakeHTTPGet(requestURL *url.URL, internalSecret string, logger *log.Entry) ([]byte, error) {
	switch requestURL.String() {
	case "http://localhost:8877/ambassador/v0/diag/?json=true":
		if internalSecret != "" {
			return nil, errors.New("Only .ambassador-internal URLs should get secret")
		}
		return testdataAmbassadorDiagJSON, nil
	case "http://ambassador/openapi/.ambassador-internal/openapi-docs":
		if internalSecret == "" {
			return nil, errors.New(".ambassador-internal URLs should get secret")
		}
		return testdataOpenAPIDocsJSON, nil
	case "http://ambassador/qotm/.ambassador-internal/openapi-docs":
		if internalSecret == "" {
			return nil, errors.New(".ambassador-internal URLs should get secret")
		}
		return []byte("<html><body>not a json</body></html>"), nil
	default:
		return nil, fmt.Errorf("Unknown URL")
	}
}

func urlMust(u *url.URL, err error) *url.URL {
	if err != nil {
		panic(err)
	}
	return u
}

// Big picture test of retrieving info from diagd and OpenAPI endpoint.
func TestFetcherRetrieve(t *testing.T) {
	g := NewGomegaWithT(t)
	s := NewServer("", nil)

	// Start out knowing about one service, but it's going to go away:
	oldSvc := kubernetes.Service{Name: "old"}
	s.AddService(oldSvc, "http://whatev", "/foo", nil)
	g.Expect(s.KnownServices()).To(Equal([]kubernetes.Service{oldSvc}))

	f := NewFetcher(s, fakeHTTPGet, s.KnownServices(), types.Config{
		AmbassadorAdminURL:    urlMust(url.Parse("http://localhost:8877")),
		AmbassadorInternalURL: urlMust(url.Parse("http://ambassador")),
		DevPortalPollInterval: 1,
		AmbassadorExternalURL: urlMust(url.Parse("https://publicapi.com")),
	})

	pprefix := &struct{ prefix string }{prefix: "--none--"}
	f.SubscribeMappingObserver("devportal_mapping", func(prefix, rewrite string) bool {
		pprefix.prefix = prefix
		return false
	})

	f.logger.Info("retrieving")
	// When we retrieve we will be told about a bunch of new services. Only
	// one of them will have OpenAPI docs, though.
	ctx, ctxCancel := context.WithCancel(context.Background())
	go f.Run(ctx)
	f.Retrieve()
	ctxCancel()

	g.Expect(pprefix.prefix).To(Equal("/docs"))

	httpbin := kubernetes.Service{Name: "httpbin", Namespace: "default"}
	devportal := kubernetes.Service{Name: "devportal", Namespace: "default"}
	_openapi := kubernetes.Service{Name: "openapi", Namespace: "default"}
	qotm := kubernetes.Service{Name: "qotm", Namespace: "default"}

	// old service went away, we detected new ones:
	knownServices := s.KnownServices()
	f.logger.Info("known services", knownServices)
	sort.Slice(knownServices, func(i, j int) bool {
		return knownServices[i].Name < knownServices[j].Name
	})
	f.logger.Info("known services (sorted)", knownServices)
	g.Expect(knownServices).To(Equal([]kubernetes.Service{devportal, httpbin, _openapi, qotm}))

	// openapi has OpenAPI doc, others don't:
	g.Expect(s.K8sStore.Get(httpbin, false)).To(Equal(&kubernetes.ServiceMetadata{
		Prefix:  "/httpbin",
		BaseURL: "https://publicapi.com", HasDoc: false, Doc: nil}))
	// This one has custom Host route in the annotation:
	g.Expect(s.K8sStore.Get(qotm, false)).To(Equal(&kubernetes.ServiceMetadata{
		Prefix:  "/qotm",
		BaseURL: "https://qotm.example.com", HasDoc: false, Doc: nil}))
	// This one has an OpenAPI doc:
	json, _ := fakeHTTPGet(urlMust(url.Parse("http://ambassador/openapi/.ambassador-internal/openapi-docs")), f.internalSecret.Get(), nil)
	g.Expect(s.K8sStore.Get(_openapi, true)).To(Equal(&kubernetes.ServiceMetadata{
		Prefix:  "/openapi",
		BaseURL: "https://publicapi.com", HasDoc: true,
		Doc: openapi.NewOpenAPI(json, "https://publicapi.com", "/openapi")}))
}
