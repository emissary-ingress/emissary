package server

import (
	"context"
	"time"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/content"
)

type ServerConfig struct {
	AmbassadorAdminURL    string
	AmbassadorInternalURL string
	AmbassadorExternalURL string
	PollFrequency         time.Duration
	ContentURL            string
}

func MakeServer(docroot string, ctx context.Context, config ServerConfig) (s *Server, err error) {

	content, err := content.NewContent(config.ContentURL)
	if err != nil {
		return
	}

	s = NewServer(docroot, content)

	knownServices := s.knownServices()
	// TODO push context into fetcher
	fetcher := NewFetcher(
		s.getServiceAdd(), s.getServiceDelete(), httpGet,
		knownServices,
		config)
	go func() {
		fetcher.retrieve()
		defer fetcher.Stop()
		<-ctx.Done()
	}()
	return
}
