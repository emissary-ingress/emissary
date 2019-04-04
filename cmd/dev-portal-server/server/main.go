package server

import (
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"time"
)

func Main(version string) {
	s := NewServer()

	serviceMap := s.K8sStore.List()
	knownServices := make([]kubernetes.Service, len(serviceMap))
	i := 0
	for k := range serviceMap {
		knownServices[i] = k
		i++
	}
	fetcher := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), knownServices,
		"http://192.168.39.80:31320", "http://192.168.39.80:31320",
		time.Second*5, "https://myapi.woot.example.com")
	defer fetcher.Stop()
	s.ServeHTTP()
}
