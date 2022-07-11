package openapi3

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
)

// ReadFromURIFunc defines a function which reads the contents of a resource
// located at a URI.
type ReadFromURIFunc func(loader *Loader, url *url.URL) ([]byte, error)

// ErrURINotSupported indicates the ReadFromURIFunc does not know how to handle a
// given URI.
var ErrURINotSupported = errors.New("unsupported URI")

// ReadFromURIs returns a ReadFromURIFunc which tries to read a URI using the
// given reader functions, in the same order. If a reader function does not
// support the URI and returns ErrURINotSupported, the next function is checked
// until a match is found, or the URI is not supported by any.
func ReadFromURIs(readers ...ReadFromURIFunc) ReadFromURIFunc {
	return func(loader *Loader, url *url.URL) ([]byte, error) {
		for i := range readers {
			buf, err := readers[i](loader, url)
			if err == ErrURINotSupported {
				continue
			} else if err != nil {
				return nil, err
			}
			return buf, nil
		}
		return nil, ErrURINotSupported
	}
}

// DefaultReadFromURI returns a caching ReadFromURIFunc which can read remote
// HTTP URIs and local file URIs.
var DefaultReadFromURI = URIMapCache(ReadFromURIs(ReadFromHTTP(http.DefaultClient), ReadFromFile))

// ReadFromHTTP returns a ReadFromURIFunc which uses the given http.Client to
// read the contents from a remote HTTP URI. This client may be customized to
// implement timeouts, RFC 7234 caching, etc.
func ReadFromHTTP(cl *http.Client) ReadFromURIFunc {
	return func(loader *Loader, location *url.URL) ([]byte, error) {
		if location.Scheme == "" || location.Host == "" {
			return nil, ErrURINotSupported
		}
		req, err := http.NewRequest("GET", location.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := cl.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode > 399 {
			return nil, fmt.Errorf("error loading %q: request returned status code %d", location.String(), resp.StatusCode)
		}
		return ioutil.ReadAll(resp.Body)
	}
}

// ReadFromFile is a ReadFromURIFunc which reads local file URIs.
func ReadFromFile(loader *Loader, location *url.URL) ([]byte, error) {
	if location.Host != "" {
		return nil, ErrURINotSupported
	}
	if location.Scheme != "" && location.Scheme != "file" {
		return nil, ErrURINotSupported
	}
	return ioutil.ReadFile(location.Path)
}

// URIMapCache returns a ReadFromURIFunc that caches the contents read from URI
// locations in a simple map. This cache implementation is suitable for
// short-lived processes such as command-line tools which process OpenAPI
// documents.
func URIMapCache(reader ReadFromURIFunc) ReadFromURIFunc {
	cache := map[string][]byte{}
	return func(loader *Loader, location *url.URL) (buf []byte, err error) {
		if location.Scheme == "" || location.Scheme == "file" {
			if !filepath.IsAbs(location.Path) {
				// Do not cache relative file paths; this can cause trouble if
				// the current working directory changes when processing
				// multiple top-level documents.
				return reader(loader, location)
			}
		}
		uri := location.String()
		var ok bool
		if buf, ok = cache[uri]; ok {
			return
		}
		if buf, err = reader(loader, location); err != nil {
			return
		}
		cache[uri] = buf
		return
	}
}
