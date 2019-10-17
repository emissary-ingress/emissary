package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/die-net/lrucache"
	"github.com/gregjones/httpcache"
	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

var httpCache = lrucache.New(8*1024, 0) // 8KiB seems like a reasonable default
var httpCacheLock sync.RWMutex

// SetHTTPCacheMaxSize adjusts the maximum size of the global HTTP
// cache used by clients returned from NewHTTPClient.  The cache size
// is measured in bytes.
//
// BUG(lukeshu): Reducing the cache maximum size to smaller than the
// current contents does not trigger an immediate cleanup; that is
// deferred until the next cache access.
func SetHTTPCacheMaxSize(n int64) {
	httpCacheLock.Lock()
	httpCache.MaxSize = n
	httpCacheLock.Unlock()
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// NewHTTPClient returns an *http.Client with several useful
// properties:
//
//  - It caches responses in a global in-memory cache, which has the
//    following properties:
//     * mostly-RFC7234-compliant
//     * resizable using the SetHTTPCacheMaxSize() function
//     * LRU eviction when it gets too big
//  - If maxStale > 0, it
//     1. Behaves as if requests set the "max-stale=${maxStale}"
//        Cache-Control directive.
//     2. Ignores "no-store" Cache-Control directive on responses (in
//        violation of RFC7234)
//     3. Ignores "no-cache" Cache-Control directive on responses (in
//        violation of RFC7234)
//  - It logs all requests+responses, and whether or not they came
//    from the network for from the cache.
func NewHTTPClient(logger types.Logger, maxStale time.Duration, insecure bool) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if maxStale > 0 {
				// Avoid mutating the request we
				// received; make a shallow copy of it
				_req := req
				req = new(http.Request)
				*req = *_req
				// and a deep copy of the Header
				req.Header = make(http.Header)
				for k, s := range _req.Header {
					req.Header[k] = s
				}
				// Set "Cache-Control: max-stale=maxStale
				cc := parseCacheControl(req.Header.Get("Cache-Control"))
				cc["max-stale"] = strconv.FormatInt(int64(maxStale.Seconds()), 10)
				req.Header.Set("Cache-Control", cc.String())
			}

			cached := true
			cacheTransport := &httpcache.Transport{
				Cache:               httpCache,
				MarkCachedResponses: false,
				Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					cached = false
					start := time.Now()
					transport := http.DefaultTransport
					if insecure {
						// this is the definition of http.DefaultTransport,
						// but with TLSClientConfig added.
						transport = &http.Transport{
							Proxy: http.ProxyFromEnvironment,
							DialContext: (&net.Dialer{
								Timeout:   30 * time.Second,
								KeepAlive: 30 * time.Second,
								DualStack: true,
							}).DialContext,
							MaxIdleConns:          100,
							IdleConnTimeout:       90 * time.Second,
							TLSHandshakeTimeout:   10 * time.Second,
							ExpectContinueTimeout: 1 * time.Second,
							// #nosec G402
							TLSClientConfig: &tls.Config{
								Renegotiation:      tls.RenegotiateOnceAsClient,
								InsecureSkipVerify: true,
							},
						}
					}
					res, err := transport.RoundTrip(req)
					dur := time.Since(start)
					if err != nil {
						err = errors.WithStack(err)
						logger.Infof("HTTP CLIENT: NET: %q %q => ERR %v (%v)", req.Method, req.URL, err, dur)
						logger.Debugf("HTTP CLIENT: stack trace: %+v", err)
					} else {
						logger.Infof("HTTP CLIENT: NET: %q %q => HTTP %d (%v)", req.Method, req.URL, res.StatusCode, dur)
					}
					if res != nil && maxStale > 0 {
						cc := parseCacheControl(res.Header.Get("Cache-Control"))
						delete(cc, "no-store")
						delete(cc, "no-cache")
						res.Header.Set("Cache-Control", cc.String())
					}
					return res, err
				}),
			}
			httpCacheLock.RLock()
			res, err := cacheTransport.RoundTrip(req)
			httpCacheLock.RUnlock()
			if cached {
				logger.Infof("HTTP CLIENT: CACHE: %q %q", req.Method, req.URL)
			}
			return res, err
		}),
	}
}

type cacheControl map[string]string

// parseCacheControl is borrowed from github.com/gregjones/httpcache
func parseCacheControl(ccHeader string) cacheControl {
	cc := cacheControl{}
	for _, part := range strings.Split(ccHeader, ",") {
		part = strings.Trim(part, " ")
		if part == "" {
			continue
		}
		if strings.ContainsRune(part, '=') {
			keyval := strings.Split(part, "=")
			cc[strings.Trim(keyval[0], " ")] = strings.Trim(keyval[1], ",")
		} else {
			cc[part] = ""
		}
	}
	return cc
}

func (cc cacheControl) String() string {
	directives := make([]string, 0, len(cc))
	for k, v := range cc {
		if v == "" {
			directives = append(directives, k)
		} else {
			directives = append(directives, k+"="+v)
		}
	}
	sort.Strings(directives)
	return strings.Join(directives, ", ")
}
