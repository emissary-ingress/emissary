package server

import (
	"net/url"
	"time"

	"github.com/datawire/apro/cmd/dev-portal-server/content"
)

func Main(
	version string, diagdURL string, ambassadorURL string, publicURL string,
	pollFrequency time.Duration, sharedSecretPath string, contentURL string) {
	url, err := url.Parse(contentURL)
	if err != nil {
		panic(err)
	}
	content, err := content.NewContent(url)
	if err != nil {
		panic(err)
	}
	s := NewServer(content)

	knownServices := s.knownServices()
	fetcher := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), httpGet, knownServices,
		diagdURL, ambassadorURL, pollFrequency, publicURL, sharedSecretPath)
	go fetcher.retrieve()
	defer fetcher.Stop()
	s.ServeHTTP()
}
