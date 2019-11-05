//go:generate go-bindata -pkg=webui -prefix=bindata/ bindata/...

package webui

import (
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

	"github.com/go-acme/lego/v3/acme"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sClientDynamic "k8s.io/client-go/dynamic"

	"github.com/datawire/apro/cmd/amb-sidecar/acmeclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
)

type firstBootWizard struct {
	staticfiles http.FileSystem
	hostsGetter k8sClientDynamic.NamespaceableResourceInterface

	snapshot atomic.Value
}

func (fb *firstBootWizard) getSnapshot() watt.Snapshot {
	return fb.snapshot.Load().(watt.Snapshot)
}

func New(
	dynamicClient k8sClientDynamic.Interface,
	snapshotCh <-chan watt.Snapshot,
) http.Handler {
	dir := os.Getenv("AES_WEBUI_BASE")
	if dir == "" {
		dir = "/ambassador/webui/bindata/"
	}
	var files http.FileSystem = http.Dir(dir)

	ret := &firstBootWizard{
		staticfiles: files,
		hostsGetter: dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "hosts"}),
	}
	ret.snapshot.Store(watt.Snapshot{})
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

func (fb *firstBootWizard) notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	file, _ := fb.staticfiles.Open("/404.html")
	io.Copy(w, file)
}

func (fb *firstBootWizard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/edge_stack/") {
		// prevent navigating to /404.html and getting a 200 response
		fb.notFound(w, r)
		return
	}
	switch r.URL.Path {
	case "/edge_stack/admin/tos-url":
		// Do this here, instead of in the web-browser,
		// because CORS.
		httpClient := httpclient.NewHTTPClient(middleware.GetLogger(r.Context()), 0, false, tls.RenegotiateNever)
		tosURL, err := getTermsOfServiceURL(httpClient, r.URL.Query().Get("ca-url"))
		if err != nil {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest, err, nil)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, tosURL)
	case "/edge_stack/admin/yaml":
		dat := &ambassadorTypesV2.Host{
			TypeMeta: &k8sTypesMetaV1.TypeMeta{
				APIVersion: "getambassador.io/v2",
				Kind:       "Host",
			},
			ObjectMeta: &k8sTypesMetaV1.ObjectMeta{
				Name:      acmeclient.NameEncode(r.URL.Query().Get("hostname")),
				Namespace: "default",
				Labels: map[string]string{
					"created-by": "aes-firstboot-web-ui",
				},
			},
			Spec: &ambassadorTypesV2.HostSpec{
				Hostname: r.URL.Query().Get("hostname"),
				AcmeProvider: &ambassadorTypesV2.ACMEProviderSpec{
					Authority: r.URL.Query().Get("acme_authority"),
					Email:     r.URL.Query().Get("acme_email"),
				},
			},
		}
		switch r.Method {
		case http.MethodGet:
			// Go ahead and fill the defaults; we won't do this when actually applying
			// it (in the http.MethodPost case), but it's informative to the user.
			acmeclient.FillDefaults(dat)
			bytes, err := yaml.Marshal(map[string]interface{}{
				"apiVersion": "getambassador.io/v2",
				"kind":       "Host",
				"metadata":   dat.ObjectMeta,
				"spec":       dat.Spec,
				"status":     dat.Status,
			})
			if err != nil {
				// We generated 'dat'; it should always be valid.
				panic(err)
			}
			// NB: YAML doesn't actually have a registered media type
			// https://www.iana.org/assignments/media-types/media-types.xhtml
			w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
			w.Write(bytes)
		case http.MethodPost:
			_, err := fb.hostsGetter.Namespace(dat.GetNamespace()).Create(&k8sTypesUnstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "getambassador.io/v2",
					"kind":       "Host",
					"metadata":   dat.ObjectMeta,
					"spec":       dat.Spec,
				},
			}, k8sTypesMetaV1.CreateOptions{})
			if err != nil {
				middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest,
					err, nil)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
				errors.New("method not allowed"), nil)
		}
	case "/edge_stack/admin/status":
		needleHostname := r.URL.Query().Get("hostname")

		var needle *ambassadorTypesV2.Host
		for _, straw := range fb.getSnapshot().Kubernetes.Host {
			if straw.GetSpec().Hostname == needleHostname {
				needle = straw
				break
			}
		}
		if needle == nil {
			io.WriteString(w, "waiting for Host resource to be created")
		} else {
			switch needle.GetStatus().GetState() {
			case ambassadorTypesV2.HostState_Initial:
				fmt.Fprintln(w,
					"state:", needle.GetStatus().GetState())
			case ambassadorTypesV2.HostState_Pending:
				fmt.Fprintln(w,
					"state:", needle.GetStatus().GetState(),
					"phaseCompleted:", needle.GetStatus().GetPhaseCompleted(),
					"phasePending:", needle.GetStatus().GetPhasePending())
			case ambassadorTypesV2.HostState_Ready:
				fmt.Fprintln(w,
					"state:", needle.GetStatus().GetState())
			case ambassadorTypesV2.HostState_Error:
				fmt.Fprintln(w,
					"state:", needle.GetStatus().GetState(),
					"phaseCompleted:", needle.GetStatus().GetPhaseCompleted(),
					"phasePending:", needle.GetStatus().GetPhasePending(),
					"error reason:", needle.GetStatus().GetReason())
			default:
				io.WriteString(w, "state: <invalid state>")
			}
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
