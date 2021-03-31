package entrypoint_test

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	http "github.com/datawire/ambassador/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/resource/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/wellknown"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/stretchr/testify/require"
)

func TestHostSemantics(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: false}, nil)

	// Figure out all the clusters we'll need.
	needClusters := []string{}

	content, err := ioutil.ReadFile("testdata/host-semantics-mappings.yaml")
	if err != nil {
		panic(err)
	}

	objs, err := kates.ParseManifests(string(content))
	if err != nil {
		panic(err)
	}

	for _, obj := range objs {
		mapping, ok := obj.(*amb.Mapping)

		if ok {
			needClusters = append(needClusters, strings.Replace(mapping.Spec.Service, "-", "_", -1))
		}
	}

	f.UpsertFile("testdata/host-semantics-hosts.yaml")
	f.UpsertFile("testdata/host-semantics-mappings.yaml")
	f.Flush()

	snap := f.GetSnapshot(HasMapping("default", "regex-authority-explicit-reject"))

	require.NotNil(t, snap)

	envoyConfig := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		for _, cluster := range needClusters {
			if FindCluster(config, ClusterNameContains(cluster)) == nil {
				return false
			}
		}

		return true
	})

	totalRoutes := 0
	goodRoutes := 0
	badRoutes := 0

	for _, l := range envoyConfig.StaticResources.Listeners {
		port := l.Address.GetSocketAddress().GetPortValue()

		fmt.Printf("LISTENER %s on port %d\n", l.Name, port)

		for _, chain := range l.FilterChains {
			fmt.Printf("  CHAIN %s\n", chain.FilterChainMatch)

			for _, filter := range chain.Filters {
				if filter.Name != wellknown.HTTPConnectionManager {
					// We only know how to create an rds listener for HttpConnectionManager
					// listeners. We must ignore all other listeners.
					continue
				}

				// Note that the hcm configuration is stored in a protobuf any, so make
				// sure that GetHTTPConnectionManager is actually returning an unmarshalled copy.
				hcm := resource.GetHTTPConnectionManager(filter)
				if hcm == nil {
					continue
				}

				// RouteSpecifier is a protobuf oneof that corresponds to the rds, route_config, and
				// scoped_routes fields. Only one of those may be set at a time.
				rs, ok := hcm.RouteSpecifier.(*http.HttpConnectionManager_RouteConfig)
				if !ok {
					continue
				}

				rc := rs.RouteConfig

				for _, vhost := range rc.VirtualHosts {
					fmt.Printf("    VHost %s\n", vhost.Name)

					for _, domain := range vhost.Domains {
						for _, route := range vhost.Routes {
							m := route.Match
							pfx := m.GetPrefix()
							hdrs := m.GetHeaders()
							scheme := "implicit-http"
							hdrSummaries := []string{}

							for _, h := range hdrs {
								hName := h.Name
								exactMatch := h.GetExactMatch()

								regexMatch := ""
								srm := h.GetSafeRegexMatch()

								if srm != nil {
									regexMatch = srm.Regex
								} else {
									regexMatch = h.GetRegexMatch()
								}

								summary := fmt.Sprintf("%#v", h)

								if exactMatch != "" {
									if hName == "x-forwarded-proto" {
										scheme = exactMatch
										continue
									}

									summary = fmt.Sprintf("%s = %s", hName, exactMatch)
								} else if regexMatch != "" {
									summary = fmt.Sprintf("%s =~ %s", hName, regexMatch)
								}

								hdrSummaries = append(hdrSummaries, summary)
							}

							renderedHdrs := ""

							if len(hdrSummaries) > 0 {
								renderedHdrs = fmt.Sprintf("\n    %s", strings.Join(hdrSummaries, "\n    "))
							}

							sep := ""

							if !strings.HasPrefix(pfx, "/") {
								sep = "/"
							}

							renderedMatch := fmt.Sprintf("%s://%s%s%s%s", scheme, domain, sep, pfx, renderedHdrs)

							// Assume we don't know WTF we want here.
							expectedAction := "PANIC"

							// If it's a secure request, always route.
							if scheme == "https" {
								expectedAction = "ROUTE"
							} else {
								// The prefix cleverly encodes the expected action.
								if strings.HasSuffix(pfx, "-route/") {
									expectedAction = "ROUTE"
								} else if strings.HasSuffix(pfx, "-redirect/") {
									expectedAction = "REDIRECT"
								} else if strings.HasSuffix(pfx, "-noaction/") {
									// This is the case where no insecure action is given, which means
									// it defaults to Redirect.
									expectedAction = "REDIRECT"
								} else {
									// This isn't good enough, since rejected routes will just not appear.
									expectedAction = "REJECT"
								}
							}

							actionRoute := route.GetRoute()
							actionRedirect := route.GetRedirect()

							finalAction := "???"
							finalActionArg := ""

							if actionRoute != nil {
								finalAction = "ROUTE"
								finalActionArg = actionRoute.GetCluster()
							} else if actionRedirect != nil {
								finalAction = "REDIRECT"

								if actionRedirect.GetHttpsRedirect() {
									finalActionArg = "HTTPS"
								} else {
									finalActionArg = fmt.Sprintf("%#v", actionRedirect)
								}
							}

							fmt.Printf("  %s\n    => %s %s\n", renderedMatch, finalAction, finalActionArg)
							totalRoutes++

							if expectedAction != finalAction {
								fmt.Printf("    !! wanted %s\n", expectedAction)
								badRoutes++
							} else {
								goodRoutes++
							}
							// require.Equal(t, expectedAction, finalAction)
						}
					}
				}
			}
		}
	}

	fmt.Printf("Total routes: %d -- good %d, bad %d\n", totalRoutes, goodRoutes, badRoutes)

	require.NotNil(t, envoyConfig)
}
