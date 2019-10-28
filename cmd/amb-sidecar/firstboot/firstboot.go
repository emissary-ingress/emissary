//go:generate go-bindata -pkg=firstboot -prefix=bindata/ -modtime=1 bindata/...

package firstboot

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/go-acme/lego/v3/acme"
	"sigs.k8s.io/yaml"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/datawire/apro/cmd/amb-sidecar/acmeclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
)

type firstBootWizard struct {
	staticfiles http.FileSystem
}

func NewFirstBootWizard() http.Handler {
	return &firstBootWizard{
		staticfiles: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""},
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

func (fb *firstBootWizard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/tos-url":
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
	case "/yaml":
		dat := &watt.Host{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "getambassador.io/v2",
				Kind:       "Host",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      acmeclient.NameEncode(r.URL.Query().Get("hostname")),
				Namespace: "default",
			},
			Spec: &ambassadorTypesV2.HostSpec{
				Hostname: r.URL.Query().Get("hostname"),
				AcmeProvider: &ambassadorTypesV2.ACMEProviderSpec{
					Authority: r.URL.Query().Get("acme_authority"),
					Email:     r.URL.Query().Get("acme_email"),
				},
			},
		}
		acmeclient.FillDefaults(dat.Spec)
		bytes, err := yaml.Marshal(dat)
		if err != nil {
			// We generated 'dat'; it should always be valid.
			panic(err)
		}
		// NB: YAML doesn't actually have a registered media type https://www.iana.org/assignments/media-types/media-types.xhtml
		w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
		w.Write(bytes)
	case "/status":
		//snapshot.GetHost(r.URL.Query().Get("host"))
		io.WriteString(w, "todo...")
	default:
		http.FileServer(fb.staticfiles).ServeHTTP(w, r)
	}
}
