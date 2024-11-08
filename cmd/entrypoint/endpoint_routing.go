package entrypoint

import (
	"context"
	"fmt"
	"net"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/ambex"
	"github.com/emissary-ingress/emissary/v3/pkg/consulwatch"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

func makeEndpoints(ctx context.Context, ksnap *snapshot.KubernetesSnapshot, consulEndpoints map[string]consulwatch.Endpoints) *ambex.Endpoints {
	k8sServices := map[string]*kates.Service{}
	for _, svc := range ksnap.Services {
		k8sServices[key(svc)] = svc
	}

	result := map[string][]*ambex.Endpoint{}

	svcEndpointSlices := map[string][]*kates.EndpointSlice{}

	// Collect all the EndpointSlices for each service if the "kubernetes.io/service-name" label is present
	for _, k8sEndpointSlice := range ksnap.EndpointSlices {
		if serviceName, labelExists := k8sEndpointSlice.Labels["kubernetes.io/service-name"]; labelExists {
			svcKey := fmt.Sprintf("%s:%s", k8sEndpointSlice.Namespace, serviceName)
			svcEndpointSlices[svcKey] = append(svcEndpointSlices[svcKey], k8sEndpointSlice)
		}
	}

	// Map each service to its corresponding endpoints from all its EndpointSlices, or fall back to Endpoints if needed
	for svcKey, svc := range k8sServices {
		if slices, ok := svcEndpointSlices[svcKey]; ok && len(slices) > 0 {
			for _, slice := range slices {
				for _, ep := range k8sEndpointSlicesToAmbex(slice, svc) {
					result[ep.ClusterName] = append(result[ep.ClusterName], ep)
				}
			}
		} else {
			// Fallback to using Endpoints if no valid EndpointSlices are available
			for _, k8sEp := range ksnap.Endpoints {
				if key(k8sEp) == svcKey {
					for _, ep := range k8sEndpointsToAmbex(k8sEp, svc) {
						result[ep.ClusterName] = append(result[ep.ClusterName], ep)
					}
				}
			}
		}
	}

	for _, consulEp := range consulEndpoints {
		for _, ep := range consulEndpointsToAmbex(ctx, consulEp) {
			result[ep.ClusterName] = append(result[ep.ClusterName], ep)
		}
	}

	return &ambex.Endpoints{Entries: result}
}

func key(resource kates.Object) string {
	return fmt.Sprintf("%s:%s", resource.GetNamespace(), resource.GetName())
}

func k8sEndpointsToAmbex(ep *kates.Endpoints, svc *kates.Service) (result []*ambex.Endpoint) {
	portmap := map[string][]string{}
	for _, p := range svc.Spec.Ports {
		port := fmt.Sprintf("%d", p.Port)
		targetPort := p.TargetPort.String()
		if targetPort == "" {
			targetPort = fmt.Sprintf("%d", p.Port)
		}

		portmap[targetPort] = append(portmap[targetPort], port)
		if p.Name != "" {
			portmap[targetPort] = append(portmap[targetPort], p.Name)
			portmap[p.Name] = append(portmap[p.Name], port)
		}
		if len(svc.Spec.Ports) == 1 {
			portmap[targetPort] = append(portmap[targetPort], "")
			portmap[""] = append(portmap[""], port)
			portmap[""] = append(portmap[""], "")
		}
	}

	for _, subset := range ep.Subsets {
		for _, port := range subset.Ports {
			if port.Protocol == kates.ProtocolTCP || port.Protocol == kates.ProtocolUDP {
				portNames := map[string]bool{}
				candidates := []string{fmt.Sprintf("%d", port.Port), port.Name, ""}
				for _, c := range candidates {
					if pns, ok := portmap[c]; ok {
						for _, pn := range pns {
							portNames[pn] = true
						}
					}
				}
				for _, addr := range subset.Addresses {
					for pn := range portNames {
						sep := "/"
						if pn == "" {
							sep = ""
						}
						result = append(result, &ambex.Endpoint{
							ClusterName: fmt.Sprintf("k8s/%s/%s%s%s", ep.Namespace, ep.Name, sep, pn),
							Ip:          addr.IP,
							Port:        uint32(port.Port),
							Protocol:    string(port.Protocol),
						})
					}
				}
			}
		}
	}

	return
}

func k8sEndpointSlicesToAmbex(endpointSlice *kates.EndpointSlice, svc *kates.Service) (result []*ambex.Endpoint) {
	portmap := map[string][]string{}
	for _, p := range svc.Spec.Ports {
		port := fmt.Sprintf("%d", p.Port)
		targetPort := p.TargetPort.String()
		if targetPort == "" {
			targetPort = fmt.Sprintf("%d", p.Port)
		}

		portmap[targetPort] = append(portmap[targetPort], port)
		if p.Name != "" {
			portmap[targetPort] = append(portmap[targetPort], p.Name)
			portmap[p.Name] = append(portmap[p.Name], port)
		}
		if len(svc.Spec.Ports) == 1 {
			portmap[targetPort] = append(portmap[targetPort], "")
			portmap[""] = append(portmap[""], port)
			portmap[""] = append(portmap[""], "")
		}
	}

	for _, endpoint := range endpointSlice.Endpoints {
		for _, port := range endpointSlice.Ports {
			if *port.Protocol == kates.ProtocolTCP || *port.Protocol == kates.ProtocolUDP {
				portNames := map[string]bool{}
				candidates := []string{fmt.Sprintf("%d", *port.Port), *port.Name, ""}
				for _, c := range candidates {
					if pns, ok := portmap[c]; ok {
						for _, pn := range pns {
							portNames[pn] = true
						}
					}
				}
				if *endpoint.Conditions.Ready {
					for _, address := range endpoint.Addresses {
						for pn := range portNames {
							sep := "/"
							if pn == "" {
								sep = ""
							}
							result = append(result, &ambex.Endpoint{
								ClusterName: fmt.Sprintf("k8s/%s/%s%s%s", svc.Namespace, svc.Name, sep, pn),
								Ip:          address,
								Port:        uint32(*port.Port),
								Protocol:    string(*port.Protocol),
							})
						}
					}
				}
			}
		}
	}

	return
}

func consulEndpointsToAmbex(ctx context.Context, endpoints consulwatch.Endpoints) (result []*ambex.Endpoint) {
	for _, ep := range endpoints.Endpoints {
		addrs, err := net.LookupHost(ep.Address)
		if err != nil {
			dlog.Errorf(ctx, "error resolving consul address %s: %+v", ep.Address, err)
			continue
		}
		for _, addr := range addrs {
			result = append(result, &ambex.Endpoint{
				ClusterName: fmt.Sprintf("consul/%s/%s", endpoints.Id, endpoints.Service),
				Ip:          addr,
				Port:        uint32(ep.Port),
				Protocol:    "TCP",
			})
		}
	}

	return
}
