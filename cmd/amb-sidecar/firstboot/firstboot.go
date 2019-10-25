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

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
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
	case "/status":
		//snapshot.GetHost(r.URL.Query().Get("host"))
		io.WriteString(w, "todo...")
	default:
		http.FileServer(fb.staticfiles).ServeHTTP(w, r)
	}
}
