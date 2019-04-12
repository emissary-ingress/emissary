package server

import (
	"time"
)

func Main(
	version string, diagdURL string, ambassadorURL string, publicURL string,
	pollFrequency time.Duration) {
	s := NewServer()

	knownServices := s.knownServices()
	fetcher := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), insecureHttpGet, knownServices,
		diagdURL, ambassadorURL, pollFrequency, publicURL)
	defer fetcher.Stop()
	s.ServeHTTP()
}
