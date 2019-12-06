//go:generate go-bindata -pkg=webui -prefix=bindata/ bindata/...

package webui

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/acme"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	k8sClientDynamic "k8s.io/client-go/dynamic"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	filtercontroller "github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	rls "github.com/datawire/apro/cmd/amb-sidecar/ratelimits"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/resourceserver/rfc6750"
)

type LoginClaimsV1 struct {
	LoginTokenVersion  string `json:"login_token_version"`
	jwt.StandardClaims `json:",inline"`
}

type Snapshot struct {
	Watt       map[string]map[string]interface{}
	Diag       json.RawMessage
	License    LicenseInfo
	RedisInUse bool
}

type LicenseInfo struct {
	Claims            *licensekeys.LicenseClaimsLatest
	HardLimit         bool
	FeaturesOverLimit []string
}

type firstBootWizard struct {
	cfg         types.Config
	staticfiles http.FileSystem
	hostsGetter k8sClientDynamic.NamespaceableResourceInterface

	snapshotStore    *watt.SnapshotStore
	rlController     *rls.RateLimitController
	filterController *filtercontroller.Controller
	limiter          limiter.Limiter
	haveRedis        bool

	privkey *rsa.PrivateKey
	pubkey  *rsa.PublicKey
}

func (fb *firstBootWizard) getSnapshot() Snapshot {
	var ret Snapshot

	if err := json.Unmarshal(fb.snapshotStore.Get().Raw, &ret.Watt); err != nil || ret.Watt == nil {
		ret.Watt = make(map[string]map[string]interface{})
	}
	// XXX we should really have watt watch everything, but for
	// now I'm just patching over that stuff here.
	if ret.Watt["Kubernetes"] == nil {
		ret.Watt["Kubernetes"] = make(map[string]interface{})
	}
	ret.Watt["Kubernetes"]["RateLimit"] = fb.rlController.GetLimits()
	ret.Watt["Kubernetes"]["Filter"] = func() []crd.Filter {
		dict := fb.filterController.LoadFilters()
		// consistent order
		qnames := make([]string, 0, len(dict))
		for qname := range dict {
			qnames = append(qnames, qname)
		}
		sort.Strings(qnames)
		// main
		list := make([]crd.Filter, 0, len(dict))
		for _, filter := range dict {
			list = append(list, filter)
		}
		return list
	}()
	ret.Watt["Kubernetes"]["FilterPolicy"], _ = fb.filterController.LoadPolicies()

	ret.Diag = func() json.RawMessage {
		resp, err := http.Get("http://127.0.0.1:8877/ambassador/v0/diag/?json=true")
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil
		}
		return json.RawMessage(bodyBytes)
	}()

	ret.License = LicenseInfo{
		Claims:            fb.limiter.GetClaims(),
		HardLimit:         fb.limiter.IsHardLimitAtPointInTime(),
		FeaturesOverLimit: fb.limiter.GetFeaturesOverLimitAtPointInTime(),
	}

	ret.RedisInUse = fb.haveRedis

	return ret
}

func New(
	cfg types.Config,
	dynamicClient k8sClientDynamic.Interface,
	snapshotStore *watt.SnapshotStore,
	rlController *rls.RateLimitController,
	filterController *filtercontroller.Controller,
	privkey *rsa.PrivateKey,
	pubkey *rsa.PublicKey,
	limiter limiter.Limiter,
	redisPool *pool.Pool,
) http.Handler {
	var files http.FileSystem = http.Dir(cfg.DevWebUIDir)

	return &firstBootWizard{
		cfg:         cfg,
		staticfiles: files,
		hostsGetter: dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "hosts"}),

		snapshotStore:    snapshotStore,
		rlController:     rlController,
		filterController: filterController,
		limiter:          limiter,
		haveRedis:        redisPool != nil,

		privkey: privkey,
		pubkey:  pubkey,
	}
}

func getTermsOfServiceURL(httpClient *http.Client, caURL string) (string, error) {
	resp, err := httpClient.Get(caURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var dir acme.Directory
	if err = json.Unmarshal(bodyBytes, &dir); err != nil {
		return "", err
	}
	return dir.Meta.TermsOfService, nil
}

func (fb *firstBootWizard) isAuthorized(r *http.Request) bool {
	now := time.Now()
	duration := -5 * time.Minute
	toleratedNow := now.Add(duration)

	nowUnix := now.Unix()
	toleratedNowUnix := toleratedNow.Unix()

	tokenString, _ := rfc6750.GetFromHeader(r.Header)
	if tokenString == "" {
		return false
	}

	var claims LoginClaimsV1

	if fb.pubkey == nil {
		dlog.GetLogger(r.Context()).Warningln("bypassing JWT validation for request")
		return true
	}
	jwtParser := jwt.Parser{ValidMethods: []string{"PS512"}}
	_, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(tokenString, &claims, func(_ *jwt.Token) (interface{}, error) {
		return fb.pubkey, nil
	}))
	if err != nil {
		return false
	}

	var expiresAtVerification = claims.VerifyExpiresAt(nowUnix, true)
	var issuedAtVerification = claims.VerifyIssuedAt(toleratedNowUnix, true)
	var notBeforeVerification = claims.VerifyNotBefore(toleratedNowUnix, true)
	var loginTokenVersionVerification = claims.LoginTokenVersion == "v1"
	if expiresAtVerification && /* issuedAtVerification && notBeforeVerification && */ loginTokenVersionVerification {
		return true
	} else {
		dlog.GetLogger(r.Context()).Warningln("token failed verification (exp,iat,nbf,vers): " +
			strconv.FormatBool(expiresAtVerification) + " " +
			strconv.FormatBool(issuedAtVerification) + " " +
			strconv.FormatBool(notBeforeVerification) + " " +
			strconv.FormatBool(loginTokenVersionVerification))
		return false
	}
}

func (fb *firstBootWizard) registerActivity(w http.ResponseWriter, r *http.Request) {
	if fb.privkey == nil {
		dlog.GetLogger(r.Context()).Warningln("bypassing JWT refesh")
		return
	}
	// Keep this in-sync with edgectl/aes_login.go
	now := time.Now()
	duration := 30 * time.Minute
	token, err := jwt.NewWithClaims(jwt.GetSigningMethod("PS512"), &LoginClaimsV1{
		"v1",
		jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			NotBefore: now.Unix(),
			ExpiresAt: (now.Add(duration)).Unix(),
		},
	}).SignedString(fb.privkey)
	if err != nil {
		dlog.GetLogger(r.Context()).Warningln("failed to generate JWT", err)
		return
	}

	// Keep this in-sync with snapshot.js:updateCredentials()
	http.SetCookie(w, &http.Cookie{
		Name:  "edge_stack_auth",
		Value: token,
		Path:  "/edge_stack/",
	})
}

func (fb *firstBootWizard) notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	file, err := fb.staticfiles.Open("/404.html")
	if err != nil {
		fmt.Fprintf(w, "<p>there was an error loading 404.html; is your <tt>DEV_WEBUI_DIR</tt> set correctly?</p>")
		return
	}
	io.Copy(w, file)
}

func (fb *firstBootWizard) forbidden(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	io.WriteString(w, "Ambassador Edge Stack admin webui API forbidden")
}

//nolint:gocyclo
func (fb *firstBootWizard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/edge_stack/") {
		// prevent navigating to /404.html and getting a 200 response
		fb.notFound(w, r)
		return
	}
	if fb.cfg.DevWebUIWebstorm != "" {
		/* When developing locally with NetBrains WebStorm, it opens Chrome at post 63342, so
		 * we need to allow Chrome to CORS request to this local go server. Chrome does pre-flight
		 * checks with the http OPTIONS, so respond appropriately to that.. */
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:63342")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		default:
			/* ..and for the http GETs and POSTs, reply with the necessary CORS header. */
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:63342")
		}
		/* Learn more about CORS: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS */
	}
	switch r.URL.Path {
	case "/edge_stack/api/tos-url":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		fb.registerActivity(w, r)
		// Do this here, instead of in the web-browser,
		// because CORS.
		httpClient := httpclient.NewHTTPClient(dlog.GetLogger(r.Context()), 0, false, tls.RenegotiateNever)
		tosURL, err := getTermsOfServiceURL(httpClient, r.URL.Query().Get("ca-url"))
		if err != nil {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest, err, nil)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, tosURL)
	case "/edge_stack/api/config/ambassador-cluster-id":
		// no authentication for this one
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, fb.cfg.AmbassadorClusterID)
	case "/edge_stack/api/config/pod-namespace":
		// no authentication for this one
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, fb.cfg.PodNamespace)
	case "/edge_stack/api/config/debug-config":
		// no authentication for this one
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "v1\n")
		io.WriteString(w, os.Getenv("DEV_WEBUI_PORT"))
		io.WriteString(w, "\n")
	case "/edge_stack/api/snapshot":
		snapshotHost := fb.cfg.DevWebUISnapshotHost
		if snapshotHost != "" {
			client := &http.Client{
				Transport: &http.Transport{
					// #nosec G402
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
			req, err := http.NewRequest("GET",
				fmt.Sprintf("https://%s/edge_stack/api/snapshot", snapshotHost),
				nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			req.Header = r.Header

			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer resp.Body.Close()

			// headers

			for name, values := range resp.Header {
				w.Header()[name] = values
			}

			// status (must come after setting headers and before copying body)
			w.WriteHeader(resp.StatusCode)

			// body
			io.Copy(w, resp.Body)
			return
		}

		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(fb.getSnapshot())
	case "/edge_stack/api/activity":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		switch r.Method {
		case http.MethodPost:
			fb.registerActivity(w, r)
		default:
			middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
				errors.New("method not allowed"), nil)
		}
	case "/edge_stack/api/apply":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		switch r.Method {
		case http.MethodPost:
			fb.registerActivity(w, r) // the happy path
		case http.MethodOptions:
			return // do nothing
		default:
			middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
				errors.New("method not allowed"), nil)
		}
		apply := supervisor.Command("WEBUI", "kubectl", "apply", "-f", "-")
		apply.Stdin = r.Body
		var output bytes.Buffer
		apply.Stdout = &output
		apply.Stderr = &output
		apply.Run()
		if apply.ProcessState.ExitCode() == 0 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Write(output.Bytes())
	case "/edge_stack/api/delete":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		switch r.Method {
		case http.MethodPost:
			fb.registerActivity(w, r) // the happy path
		case http.MethodOptions:
			return // do nothing
		default:
			middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
				errors.New("method not allowed"), nil)
		}
		decoder := json.NewDecoder(r.Body)
		var obj struct {
			Namespace string
			Names     []string
		}
		err := decoder.Decode(&obj)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, err.Error())
			return
		}
		delete := supervisor.Command("WEBUI", "kubectl",
			append([]string{"delete", "--wait=false", "--namespace", obj.Namespace}, obj.Names...)...)
		var output bytes.Buffer
		delete.Stdout = &output
		delete.Stderr = &output
		delete.Run()
		if delete.ProcessState.ExitCode() == 0 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Write(output.Bytes())
	case "/edge_stack/api/log-level":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		fb.registerActivity(w, r)
		switch r.Method {
		case http.MethodPost:
			query := make(url.Values)
			query.Set("loglevel", r.FormValue("loglevel"))
			resp, err := http.Get("http://127.0.0.1:8877/ambassador/v0/diag/?" + query.Encode())
			if err != nil {
				middleware.ServeErrorResponse(w, r.Context(), http.StatusBadGateway,
					err, nil)
				return
			}
			resp.Body.Close()
			w.WriteHeader(resp.StatusCode)
		default:
			middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
				errors.New("method not allowed"), nil)
		}
	default:
		var fi os.FileInfo

		// OK. Is this a directory with an index.html in it?
		cleaned := path.Clean(r.URL.Path)
		indexPath := path.Join(cleaned, "index.html")

		openFile, err := fb.staticfiles.Open(indexPath)

		if err != nil {
			// Nope. Can we open it at all?

			openFile, err = fb.staticfiles.Open(cleaned)

			if err == nil {
				// Yup. Is it a directory?
				fi, err = openFile.Stat()

				if err == nil {
					if fi.IsDir() {
						// Yup. Force an error so we don't serve it.
						err = errors.New("is directory")
					}
				}
			}
		}

		if openFile != nil {
			openFile.Close()
		}

		if err != nil { // was if os.IsNotExist(err), but why limit it?
			// use our custom 404 handler instead of http.FileServer's
			fb.notFound(w, r)
			return
		}

		http.FileServer(fb.staticfiles).ServeHTTP(w, r)
	}
}
