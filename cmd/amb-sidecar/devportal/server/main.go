package server

import (
	"context"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

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
		s.getServiceAdd(), s.getServiceDelete(), httpGet, knownServices,
		config.AmbassadorAdminURL, config.AmbassadorInternalURL,
		config.PollFrequency, config.AmbassadorExternalURL)
	go func() {
		fetcher.retrieve()
		defer fetcher.Stop()
		<-ctx.Done()
	}()
	return
}

func Main(
	version string, ambassadorAdminURL string, ambassadorInternalURL string, ambassadorExternalURL string,
	pollFrequency time.Duration, contentURL string) {

	config := ServerConfig{
		AmbassadorAdminURL:    ambassadorAdminURL,
		AmbassadorInternalURL: ambassadorInternalURL,
		AmbassadorExternalURL: ambassadorExternalURL,
		PollFrequency:         pollFrequency,
		ContentURL:            contentURL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := MakeServer("", ctx, config)
	if err != nil {
		panic(err)
	}

	log.Fatal(http.ListenAndServe("0.0.0.0:8680", s.Router()))
}
