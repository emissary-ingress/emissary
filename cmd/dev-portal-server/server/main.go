package server

import (
	"time"

	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
)

func Main(
	version string, diagdURL string, ambassadorURL string, publicURL string,
	pollFrequency time.Duration) {
	s := NewServer()

	serviceMap := s.K8sStore.List()
	knownServices := make([]kubernetes.Service, len(serviceMap))
	i := 0
	for k := range serviceMap {
		knownServices[i] = k
		i++
	}
	fetcher := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), httpGet, knownServices,
		diagdURL, ambassadorURL, pollFrequency, publicURL)
	defer fetcher.Stop()
	s.ServeHTTP()
}
