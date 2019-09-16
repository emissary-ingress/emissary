package server

import (
	"context"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/content"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func MakeServer(docroot string, ctx context.Context, config types.Config) (s *Server, err error) {

	content, err := content.NewContent(config.DevPortalContentURL)
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
