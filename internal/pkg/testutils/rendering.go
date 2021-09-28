package testutils

import (
	"fmt"
	"sort"
	"strings"

	bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	http "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"
	"github.com/datawire/ambassador/v2/pkg/kates"
)

type RenderedRoute struct {
	Scheme         string `json:"scheme"`
	Host           string `json:"host"`
	Path           string `json:"path"`
	Authority      string `json:"authority"`
	AuthorityMatch string `json:"authorityMatch"`
	Action         string `json:"action"`
	ActionArg      string `json:"action_arg"`
}

func (rr *RenderedRoute) String() string {
	s := fmt.Sprintf("%s%s: %s://%s%s", rr.Action, rr.ActionArg, rr.Scheme, rr.Host, rr.Path)

	if rr.Authority != "" {
		s += fmt.Sprintf(" (:authority %s %s)", rr.AuthorityMatch, rr.Authority)
	}

	return s
}

type RenderedVHost struct {
	Name   string          `json:"name"`
	Routes []RenderedRoute `json:"routes"`
}

func (rvh *RenderedVHost) AddRoute(rr RenderedRoute) {
	rvh.Routes = append(rvh.Routes, rr)
}

func NewRenderedVHost(name string) RenderedVHost {
	return RenderedVHost{
		Name:   name,
		Routes: []RenderedRoute{},
	}
}

type RenderedChain struct {
	ServerNames       []string                  `json:"server_names"`
	TransportProtocol string                    `json:"transport_protocol"`
	VHosts            map[string]*RenderedVHost `json:"-"`
	VHostList         []*RenderedVHost          `json:"vhosts"`
}

func (rchain *RenderedChain) AddVHost(rvh *RenderedVHost) {
	rchain.VHosts[rvh.Name] = rvh
}

func (rchain *RenderedChain) GetVHost(vhostname string) *RenderedVHost {
	return rchain.VHosts[vhostname]
}

func NewRenderedChain(serverNames []string, transportProtocol string) RenderedChain {
	chain := RenderedChain{
		ServerNames:       nil,
		TransportProtocol: transportProtocol,
		VHosts:            map[string]*RenderedVHost{},
		VHostList:         []*RenderedVHost{},
	}

	if len(serverNames) > 0 {
		chain.ServerNames = []string{}

		for _, name := range serverNames {
			if (name != "") && (name != "*") {
				chain.ServerNames = append(chain.ServerNames, name)
			}
		}
	}

	return chain
}

type RenderedListener struct {
	Name      string                    `json:"name"`
	Port      uint32                    `json:"port"`
	Chains    map[string]*RenderedChain `json:"-"`
	ChainList []*RenderedChain          `json:"chains"`
}

func (rl *RenderedListener) AddChain(rchain *RenderedChain) {
	hostname := "*"

	if len(rchain.ServerNames) > 0 {
		hostname = rchain.ServerNames[0]
	}

	xport := rchain.TransportProtocol

	extant := rl.GetChain(hostname, xport)

	if extant != nil {
		panic(fmt.Errorf("chain for %s, %s already exists in %s", hostname, xport, rl.Name))
	}

	key := fmt.Sprintf("%s-%s", hostname, xport)

	rl.Chains[key] = rchain
}

func (rl *RenderedListener) GetChain(hostname string, xport string) *RenderedChain {
	key := fmt.Sprintf("%s-%s", hostname, xport)

	return rl.Chains[key]
}

func NewRenderedListener(name string, port uint32) RenderedListener {
	return RenderedListener{
		Name:      name,
		Port:      port,
		Chains:    map[string]*RenderedChain{},
		ChainList: []*RenderedChain{},
	}
}

func NewAmbassadorListener(port uint32) RenderedListener {
	return RenderedListener{
		Name:   fmt.Sprintf("ambassador-listener-0.0.0.0-%d", port),
		Port:   port,
		Chains: map[string]*RenderedChain{},
	}
}

func NewAmbassadorMapping(name string, pfx string) v3alpha1.Mapping {
	return v3alpha1.Mapping{
		TypeMeta:   kates.TypeMeta{Kind: "Mapping"},
		ObjectMeta: kates.ObjectMeta{Namespace: "default", Name: name},
		Spec: v3alpha1.MappingSpec{
			Prefix:  pfx,
			Service: "127.0.0.1:8877",
		},
	}
}

func JSONifyRenderedListeners(renderedListeners []RenderedListener) string {
	// Why is this needed? JSONifying renderedListeners directly always
	// shows empty listeners -- kinda feels like something's getting copied
	// in a way I'm not awake enough to follow right now.
	toDump := []RenderedListener{}

	for _, l := range renderedListeners {
		for _, c := range l.Chains {
			for _, v := range c.VHosts {
				if len(v.Routes) > 1 {
					sort.SliceStable(v.Routes, func(i, j int) bool {
						if v.Routes[i].Path != v.Routes[j].Path {
							return v.Routes[i].Path < v.Routes[j].Path
						}

						if v.Routes[i].Host != v.Routes[j].Host {
							return v.Routes[i].Host < v.Routes[j].Host
						}

						if v.Routes[i].Action != v.Routes[j].Action {
							return v.Routes[i].Action < v.Routes[j].Action
						}

						return v.Routes[i].ActionArg < v.Routes[j].ActionArg
					})
				}

				c.VHostList = append(c.VHostList, v)
			}

			if len(c.VHostList) > 1 {
				sort.SliceStable(c.VHostList, func(i, j int) bool {
					return c.VHostList[i].Name < c.VHostList[j].Name
				})
			}

			l.ChainList = append(l.ChainList, c)
		}

		if len(l.ChainList) > 1 {
			sort.SliceStable(l.ChainList, func(i, j int) bool {
				sNamesI := l.ChainList[i].ServerNames
				sNamesJ := l.ChainList[j].ServerNames

				if (len(sNamesI) > 0) && (len(sNamesJ) > 0) {
					if l.ChainList[i].ServerNames[0] != l.ChainList[j].ServerNames[0] {
						return l.ChainList[i].ServerNames[0] < l.ChainList[j].ServerNames[0]
					}
				}

				return l.ChainList[i].TransportProtocol < l.ChainList[j].TransportProtocol
			})
		}

		toDump = append(toDump, l)
	}

	if len(toDump) > 1 {
		sort.SliceStable(toDump, func(i, j int) bool {
			return toDump[i].Port < toDump[j].Port
		})
	}

	return JSONify(toDump)
}

type Candidate struct {
	Scheme    string
	Action    string
	ActionArg string
}

func RenderEnvoyConfig(envoyConfig *bootstrap.Bootstrap) []RenderedListener {
	renderedListeners := make([]RenderedListener, 0, 2)

	for _, l := range envoyConfig.StaticResources.Listeners {
		port := l.Address.GetSocketAddress().GetPortValue()

		fmt.Printf("LISTENER %s on port %d (chains %d)\n", l.Name, port, len(l.FilterChains))
		rlistener := NewRenderedListener(l.Name, port)

		for _, chain := range l.FilterChains {
			fmt.Printf("  CHAIN %s\n", chain.FilterChainMatch)

			rchain := NewRenderedChain(chain.FilterChainMatch.ServerNames, chain.FilterChainMatch.TransportProtocol)

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

					rvh := NewRenderedVHost(vhost.Name)

					for _, domain := range vhost.Domains {
						for _, route := range vhost.Routes {
							m := route.Match
							pfx := m.GetPrefix()
							hdrs := m.GetHeaders()
							scheme := "implicit-http"

							if !strings.HasPrefix(pfx, "/") {
								pfx = "/" + pfx
							}

							authority := ""
							authorityMatch := ""

							for _, h := range hdrs {
								hName := h.Name
								prefixMatch := h.GetPrefixMatch()
								suffixMatch := h.GetSuffixMatch()
								exactMatch := h.GetExactMatch()

								regexMatch := ""
								srm := h.GetSafeRegexMatch()

								if srm != nil {
									regexMatch = srm.Regex
									// } else {
									// 	regexMatch = h.GetRegexMatch()
								}

								// summary := fmt.Sprintf("%#v", h)

								if exactMatch != "" {
									if hName == "x-forwarded-proto" {
										scheme = exactMatch
										continue
									}

									authority = exactMatch
									authorityMatch = "=="
								} else if prefixMatch != "" {
									authority = prefixMatch + "*"
									authorityMatch = "gl~"
								} else if suffixMatch != "" {
									authority = "*" + suffixMatch
									authorityMatch = "gl~"
								} else if regexMatch != "" {
									authority = regexMatch
									authorityMatch = "re~"
								}
							}

							actionRoute := route.GetRoute()
							actionRedirect := route.GetRedirect()

							finalAction := "???"
							finalActionArg := ""

							if actionRoute != nil {
								finalAction = "ROUTE"
								finalActionArg = " " + actionRoute.GetCluster()
							} else if actionRedirect != nil {
								finalAction = "REDIRECT"

								if actionRedirect.GetHttpsRedirect() {
									finalActionArg = " HTTPS"
								} else {
									finalActionArg = fmt.Sprintf(" %#v", actionRedirect)
								}
							}

							rroute := RenderedRoute{
								Scheme:         scheme,
								Host:           domain,
								Path:           pfx,
								Authority:      authority,
								AuthorityMatch: authorityMatch,
								Action:         finalAction,
								ActionArg:      finalActionArg,
							}

							rvh.AddRoute(rroute)

							fmt.Printf("      %s\n", rroute.String())

							// if expectedAction != finalAction {
							// 	fmt.Printf("    !! wanted %s\n", expectedAction)
							// 	badRoutes++
							// } else {
							// 	goodRoutes++
							// }
							// require.Equal(t, expectedAction, finalAction)
						}
					}

					rchain.AddVHost(&rvh)
				}
			}

			rlistener.AddChain(&rchain)
		}

		renderedListeners = append(renderedListeners, rlistener)
	}

	return renderedListeners
}
