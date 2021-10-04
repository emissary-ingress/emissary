package entrypoint

import (
	"context"
	"fmt"
	"net"

	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

func makeEndpoints(ctx context.Context, ksnap *snapshotTypes.KubernetesSnapshot, consulEndpoints map[string]consulwatch.Endpoints) *ambex.Endpoints {
	k8sServices := map[string]*kates.Service{}
	for _, svc := range ksnap.Services {
		k8sServices[key(svc)] = svc
	}

	result := map[string][]*ambex.Endpoint{}

	for _, k8sEp := range ksnap.Endpoints {
		svc, ok := k8sServices[key(k8sEp)]
		if !ok {
			continue
		}
		for _, ep := range k8sEndpointsToAmbex(k8sEp, svc) {
			result[ep.ClusterName] = append(result[ep.ClusterName], ep)
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
