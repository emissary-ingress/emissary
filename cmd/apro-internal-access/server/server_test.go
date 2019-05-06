package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datawire/apro/cmd/apro-internal-access/secret"
	. "github.com/onsi/gomega"
)

// If the correct shared secret is given in X-Ambassador-Internal-Auth header,
// access is allowed, otherwise it is denied.
func TestExtauth(t *testing.T) {
	g := NewGomegaWithT(t)
	file, _ := ioutil.TempFile("", "prefix")
	file.WriteString("abc")
	theSecret := secret.LoadSecret(file.Name())
	s := NewServer(file.Name())

	// No authentication info:
	req, err := http.NewRequest(
		"GET", "/extauth/foo/.ambassador-internal/bar", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusUnauthorized))

	// Wrong authentication info:
	req, err = http.NewRequest(
		"GET", "/extauth/foo/.ambassador-internal/bar", nil)
	req.Header.Set("X-Ambassador-Internal-Auth", "wrong secret")
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusUnauthorized))

	// Correct authentication info:
	req, err = http.NewRequest(
		"GET", "/extauth/foo/.ambassador-internal/bar", nil)
	req.Header.Set("X-Ambassador-Internal-Auth", theSecret)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusOK))
}
