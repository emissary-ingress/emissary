package util

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Error is the default error response object ..
type Error struct {
	Message string `json:"message"`
}

// HTTPContextKey is a helper to define an HTTP request context key.
type HTTPContextKey string

func (c HTTPContextKey) String() string {
	return string(c)
}

// ToJSONResponse takes the HTTP response writer object, the status code, a json struct and
// sets the writer to produce a json response.
func ToJSONResponse(w http.ResponseWriter, status int, i interface{}) {
	jsonResponse, err := json.Marshal(i)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "Application/Json")
	w.WriteHeader(status)
	w.Write(jsonResponse)
}

// OriginalURL(r) is like r.URL, but obeys `Host` and
// `X-Forwarded-Proto`.
//
// TODO(lukeshu): Use RFC 7239 `Forwarded` instead of the old
// non-standard `X-Forwarded-Proto`.
func OriginalURL(r *http.Request) *url.URL {
	u, _ := r.URL.Parse("")
	u.Host = r.Host
	if r.TLS != nil || r.Header.Get("x-forwarded-proto") == "https" {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	return u
}

// ContextualRoundTripper provides a way to make HTTP requests that carry some
// header context from an incoming request, the origin.
type ContextualRoundTripper struct {
	Origin  *http.Request
	Headers []string
	Inner   http.RoundTripper
}

// RoundTrip copies the relevant headers into the client request
func (crt *ContextualRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, header := range crt.Headers {
		req.Header.Set(header, crt.Origin.Header.Get(header))
	}
	return crt.Inner.RoundTrip(req)
}

// NewHeaderPassingClient yields an HTTP client that passes along the specified
// headers from the origin request.
func NewHeaderPassingClient(origin *http.Request, headers []string) http.Client {
	crt := ContextualRoundTripper{
		Origin:  origin,
		Headers: headers,
		Inner:   http.DefaultTransport,
	}
	client := http.Client{
		Transport: &crt,
	}
	return client
}

// Function signature to pass to GetBodyBytes and GetBodyJSON to check response status.
// A nil value means no check is performed. body data is passed in mostly for logging purposes.
type ResponseStatusChecker func(*http.Response, []byte) error

// Default check used by GetBodyBytes and GetBodyJSON when no checks are specified
func CheckStatusOk(res *http.Response, data []byte) error {
	if res.StatusCode != http.StatusOK {
		return errors.New("Request failed")
	}
	return nil
}

type SimpleClient struct {
	*http.Client
}

var defaultStatusChecker = []ResponseStatusChecker{CheckStatusOk}

// Return response body bytes of the GET request at specified url.
// You can optionally provide a check for the response; the default is to check
// that status was http.OK. The response needs to be of reasonable size for the
// call to succeed
func (client *SimpleClient) GetBodyBytes(url string, optionalCheck ...ResponseStatusChecker) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return client.DoBodyBytes(request, optionalCheck...)
}

// Return response body bytes of the specified request.
// You can optionally provide a check for the response; the default is to check
// that status was http.OK. The response needs to be of reasonable size for the
// call to succeed
func (client *SimpleClient) DoBodyBytes(request *http.Request, optionalCheck ...ResponseStatusChecker) ([]byte, error) {
	res, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if len(optionalCheck) == 0 {
		optionalCheck = defaultStatusChecker
	}

	data, err := readBodyBytes(res)
	if err != nil {
		return nil, err
	}

	for _, check := range optionalCheck {
		if check == nil {
			continue
		}
		err := check(res, data)
		if err != nil {
			return data, err
		}
	}

	return data, nil
}

func readBodyBytes(res *http.Response) ([]byte, error) {
	// TODO: read only a finite amount
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read HTTP response body")
	}
	return buf, nil
}

// Deserialize response body as JSON into the provided object and
// properly dispose of the response. You can optionally provide a check for the
// response; the default is to check that status was http.OK. The response needs
// to be of reasonable size for the call to succeed
func (client *SimpleClient) GetBodyJSON(url string, obj interface{}, optionalCheck ...ResponseStatusChecker) error {
	buf, err := client.GetBodyBytes(url, optionalCheck...)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, obj)
}

// Deserialize response body as JSON into the provided object and
// properly dispose of the response. You can optionally provide a check for the
// response; the default is to check that status was http.OK. The response needs
// to be of reasonable size for the call to succeed
func (client *SimpleClient) DoBodyJSON(request *http.Request, obj interface{}, optionalCheck ...ResponseStatusChecker) error {
	buf, err := client.DoBodyBytes(request, optionalCheck...)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, obj)
}
