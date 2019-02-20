package httpclient

import (
	"net/http"
	"sync"
	"time"

	"github.com/die-net/lrucache"
	"github.com/gregjones/httpcache"

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
// properties
//
//  - It caches responses in a global in-memory cache, which has the
//    following properties:
//     * mostly-RFC7234-compliant
//     * resizable using the SetHTTPCacheMaxSize() function
//     * LRU eviction when it gets too big
//  - It logs all requests+responses, and whether or not they came
//    from the network for from the cache.
func NewHTTPClient(logger types.Logger) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			cached := true
			cacheTransport := &httpcache.Transport{
				Cache:               httpCache,
				MarkCachedResponses: false,
				Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					cached = false
					start := time.Now()
					res, err := http.DefaultTransport.RoundTrip(req)
					dur := time.Since(start)
					if err != nil {
						logger.Infof("HTTP CLIENT: NET: %s %s => ERR %v (%v)", req.Method, req.URL, err, dur)
					} else {
						logger.Infof("HTTP CLIENT: NET: %s %s => HTTP %d (%v)", req.Method, req.URL, res.StatusCode, dur)
					}
					return res, err
				}),
			}
			httpCacheLock.RLock()
			res, err := cacheTransport.RoundTrip(req)
			httpCacheLock.RUnlock()
			if cached {
				logger.Infof("HTTP CLIENT: CACHE: %s %s", req.Method, req.URL)
			}
			return res, err
		}),
	}
}
