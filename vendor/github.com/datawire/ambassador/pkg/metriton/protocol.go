package metriton

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// Report is a telemetry report to submit to Metriton.
//
// See: https://github.com/datawire/metriton/blob/master/metriton/scout/jsonschema.py
type Report struct {
	Application string                 `json:"application"` // (required) The name of the application reporting the event
	InstallID   string                 `json:"install_id"`  // (required) Application installation ID (usually a UUID, but technically an opaque string)
	Version     string                 `json:"version"`     // (required) Application version number
	Metadata    map[string]interface{} `json:"metadata"`    // (optional) Additional metadata about the application
}

// Send the report to the given Metriton endpoint using the given
// httpClient.
//
// The returned *Response may be nil even if there is no error, if
// Metriton has not yet been configured to know about the Report's
// `.Application`; i.e. a Response is only returned for known
// applications.
func (r Report) Send(ctx context.Context, httpClient *http.Client, endpoint string) (*Response, error) {
	body, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(respBytes) == 0 {
		// not a recognized .Application
		return nil, nil
	}

	var parsedResp Response
	if err := json.Unmarshal(respBytes, &parsedResp); err != nil {
		return nil, err
	}
	return &parsedResp, nil
}

// Response is a response from Metriton, after submitting a Report to
// it.
type Response struct {
	AppInfo

	// Only set for .Application=="aes"
	HardLimit bool `json:"hard_limit"`

	// Disable submitting any more telemetry for the remaining
	// lifetime of this process.
	//
	// This way, if we ever make another release that turns out to
	// effectively DDoS Metriton, we can adjust the Metriton
	// server's `api.py:handle_report()` to be able to tell the
	// offending processes to shut up.
	DisableScout bool `json:"disable_scout"`
}

// AppInfo is the information that Metriton knows about an
// application.
//
// There isn't really an otherwise fixed schema for this; Metriton
// returns whatever it reads from
// f"s3://scout-datawire-io/{report.application}/app.json".  However,
// looking at all of the existing app.json files, they all agree on
// the schema
type AppInfo struct {
	Application   string   `json:"application"`
	LatestVersion string   `json:"latest_version"`
	Notices       []Notice `json:"notices"`
}

// Notice is a notice that should be displayed to the user.
//
// I have no idea what the schema for Notice is, there are none
// currently, and reverse-engineering it from what diagd.py consumes
// isn't worth the effort at this time.
type Notice interface{}
