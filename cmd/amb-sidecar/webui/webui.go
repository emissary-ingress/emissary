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
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/acme"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	k8sClientDynamic "k8s.io/client-go/dynamic"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/resourceserver/rfc6750"
)

type LoginClaimsV1 struct {
	LoginTokenVersion  string `json:"login_token_version"`
	jwt.StandardClaims `json:",inline"`
}

type firstBootWizard struct {
	cfg         types.Config
	staticfiles http.FileSystem
	hostsGetter k8sClientDynamic.NamespaceableResourceInterface

	privkey *rsa.PrivateKey
	pubkey  *rsa.PublicKey

	snapshot atomic.Value
}

func (fb *firstBootWizard) getSnapshot() watt.Snapshot {
	return fb.snapshot.Load().(watt.Snapshot)
}

func New(
	cfg types.Config,
	dynamicClient k8sClientDynamic.Interface,
	snapshotCh <-chan watt.Snapshot,
	privkey *rsa.PrivateKey,
	pubkey *rsa.PublicKey,
) http.Handler {
	var files http.FileSystem = http.Dir(cfg.DevWebUIDir)

	ret := &firstBootWizard{
		cfg:         cfg,
		staticfiles: files,
		hostsGetter: dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "hosts"}),
		privkey:     privkey,
		pubkey:      pubkey,
	}
	ret.snapshot.Store(watt.Snapshot{Raw: []byte("{}\n")})
	go func() {
		for snapshot := range snapshotCh {
			ret.snapshot.Store(snapshot)
		}
	}()
	return ret
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
	now := time.Now().Unix()

	tokenString := rfc6750.GetFromHeader(r.Header)
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

	return claims.VerifyExpiresAt(now, true) &&
		claims.VerifyIssuedAt(now, true) &&
		claims.VerifyNotBefore(now, true) &&
		claims.LoginTokenVersion == "v1"
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
	case "/edge_stack/api/snapshot":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(fb.getSnapshot().Raw)
	case "/edge_stack/api/apply":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
		fb.registerActivity(w, r)
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
		fb.registerActivity(w, r)
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
	case "/edge_stack/tls/api/empty":
		if !fb.isAuthorized(r) {
			fb.forbidden(w, r)
			return
		}
	default:
		if _, err := fb.staticfiles.Open(path.Clean(r.URL.Path)); os.IsNotExist(err) {
			// use our custom 404 handler instead of http.FileServer's
			fb.notFound(w, r)
			return
		}
		http.FileServer(fb.staticfiles).ServeHTTP(w, r)
	}
}
