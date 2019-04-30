package server

import (
	"time"
)

func Main(
	version string, diagdURL string, ambassadorURL string, publicURL string,
	pollFrequency time.Duration, sharedSecretPath string) {
	s := NewServer()

	knownServices := s.knownServices()
	fetcher := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), httpGet, knownServices,
		diagdURL, ambassadorURL, pollFrequency, publicURL, sharedSecretPath)
	defer fetcher.Stop()
	s.ServeHTTP()
}
