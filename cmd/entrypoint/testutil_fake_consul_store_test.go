package entrypoint

import (
	"sync"

	"github.com/emissary-ingress/emissary/v3/pkg/consulwatch"
)

type ConsulStore struct {
	mutex     sync.Mutex
	endpoints map[ConsulKey]consulwatch.Endpoints
}

type ConsulKey struct {
	datacenter string
	service    string
}

func NewConsulStore() *ConsulStore {
	return &ConsulStore{endpoints: map[ConsulKey]consulwatch.Endpoints{}}
}

func (c *ConsulStore) ConsulEndpoint(datacenter, service, address string, port int, tags ...string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := ConsulKey{datacenter, service}
	ep, ok := c.endpoints[key]
	if !ok {
		ep = consulwatch.Endpoints{
			Id:      datacenter,
			Service: service,
		}
	}
	ep.Endpoints = append(ep.Endpoints, consulwatch.Endpoint{
		ID:      datacenter,
		Service: service,
		Address: address,
		Port:    port,
		Tags:    tags,
	})
	c.endpoints[key] = ep
}

func (c *ConsulStore) Get(datacenter, service string) (consulwatch.Endpoints, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ep, ok := c.endpoints[ConsulKey{datacenter, service}]
	return ep, ok
}
