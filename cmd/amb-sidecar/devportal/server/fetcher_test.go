package server

import (
	"fmt"
	"io/ioutil"
	"sort"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	. "github.com/datawire/apro/cmd/amb-sidecar/devportal/kubernetes"
	. "github.com/datawire/apro/cmd/amb-sidecar/devportal/openapi"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

var testdataAmbassadorDiagJSON, _ = ioutil.ReadFile("testdata/ambassador-diag.json")
var testdataOpenAPIDocsJSON, _ = ioutil.ReadFile("testdata/openapi-docs.json")

func TestDiffCalculator(t *testing.T) {
	g := NewGomegaWithT(t)
	A, B := Service{Name: "a"}, Service{Name: "b"}
	C, D := Service{Name: "c"}, Service{Name: "d"}

	// Starting point: we know about A and B
	calc := NewDiffCalculator([]Service{A, B})

	// Round 1: we detect A and C. That means B should be marked as deleted.
	calc.Add(A)
	calc.Add(C)
	g.Expect(calc.NewRound()).To(Equal([]Service{B}))

	// Round 2: we detect A and C. That means no deletes.
	calc.Add(A)
	calc.Add(C)
	g.Expect(calc.NewRound()).To(Equal([]Service{}))

	// Round 3: we detect A and C and D. That means no deletes.
	calc.Add(A)
	calc.Add(C)
	calc.Add(D)
	g.Expect(calc.NewRound()).To(Equal([]Service{}))

	// Round 4: we detect B and C. That means A and D are deleted.
	calc.Add(B)
	calc.Add(C)
	result := calc.NewRound()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	g.Expect(result).To(Equal([]Service{A, D}))

	// Round 5: we detect nothing. That means B and C are deleted.
	result = calc.NewRound()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	g.Expect(result).To(Equal([]Service{B, C}))
}

// Hard-code diagd output, as well as OpenAPI docs for one service:
func fakeHTTPGet(url string, internalSecret string, logger *log.Entry) ([]byte, error) {
	if url == "http://localhost:8877/ambassador/v0/diag/?json=true" {
		if internalSecret != "" {
			return nil, errors.New("Only .ambassador-internal URLs should get secret")
		}
		return testdataAmbassadorDiagJSON, nil
	}
	if url == "http://ambassador/openapi/.ambassador-internal/openapi-docs" {
		if internalSecret == "" {
			return nil, errors.New(".ambassador-internal URLs should get secret")
		}
		return testdataOpenAPIDocsJSON, nil
	}
	if url == "http://ambassador/qotm/.ambassador-internal/openapi-docs" {
		if internalSecret == "" {
			return nil, errors.New(".ambassador-internal URLs should get secret")
		}
		return []byte("<html><body>not a json</body></html>"), nil
	}
	return nil, fmt.Errorf("Unknown URL")
}

// Big picture test of retrieving info from diagd and OpenAPI endpoint.
func TestFetcherRetrieve(t *testing.T) {
	g := NewGomegaWithT(t)
	s := NewServer("", nil)

	// Start out knowing about one service, but it's going to go away:
	oldSvc := Service{Name: "old"}
	s.getServiceAdd()(oldSvc, "http://whatev", "/foo", nil)
	g.Expect(s.knownServices()).To(Equal([]Service{oldSvc}))

	f := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), fakeHTTPGet,
		s.knownServices(),
		types.PortalConfig{
			AmbassadorAdminURL:    "http://localhost:8877",
			AmbassadorInternalURL: "http://ambassador",
			PollFrequency:         1,
			AmbassadorExternalURL: "https://publicapi.com",
		})

	f.logger.Info("retrieving")
	// When we retrieve we will be told about a bunch of new services. Only
	// one of them will have OpenAPI docs, though.
	f.retrieve()

	httpbin := Service{Name: "httpbin", Namespace: "default"}
	devportal := Service{Name: "devportal", Namespace: "default"}
	openapi := Service{Name: "openapi", Namespace: "default"}
	qotm := Service{Name: "qotm", Namespace: "default"}

	// old service went away, we detected new ones:
	knownServices := s.knownServices()
	f.logger.Info("known services", knownServices)
	sort.Slice(knownServices, func(i, j int) bool {
		return knownServices[i].Name < knownServices[j].Name
	})
	f.logger.Info("known services (sorted)", knownServices)
	g.Expect(knownServices).To(Equal([]Service{devportal, httpbin, openapi, qotm}))

	// openapi has OpenAPI doc, others don't:
	g.Expect(s.K8sStore.Get(httpbin, false)).To(Equal(&ServiceMetadata{
		Prefix:  "/httpbin",
		BaseURL: "https://publicapi.com", HasDoc: false, Doc: nil}))
	// This one has custom Host route in the annotation:
	g.Expect(s.K8sStore.Get(qotm, false)).To(Equal(&ServiceMetadata{
		Prefix:  "/qotm",
		BaseURL: "https://qotm.example.com", HasDoc: false, Doc: nil}))
	// This one has an OpenAPI doc:
	json, _ := fakeHTTPGet("http://ambassador/openapi/.ambassador-internal/openapi-docs", f.internalSecret.Get(), nil)
	g.Expect(s.K8sStore.Get(openapi, true)).To(Equal(&ServiceMetadata{
		Prefix:  "/openapi",
		BaseURL: "https://publicapi.com", HasDoc: true,
		Doc: NewOpenAPI(json, "https://publicapi.com", "/openapi")}))
}
