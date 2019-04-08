package server

import (
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"time"
)

func Main(version string, diagdURL string, ambassadorURL string, publicURL string) {
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
		diagdURL, ambassadorURL, time.Second*5, publicURL)
	defer fetcher.Stop()
	s.ServeHTTP()
}
