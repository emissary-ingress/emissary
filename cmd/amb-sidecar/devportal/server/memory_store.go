package server

import (
	"fmt"
	"sort"
	"sync"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/openapi"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
)

type Service struct {
	Name      string
	Namespace string
}

type ServiceMetadata struct {
	// URL prefix, e.g. /widgets
	Prefix string
	// Base URL for service e.g. https://api.example.com
	BaseURL string
	HasDoc  bool
	// May be nil even if HasDoc is true if loaded without the doc.
	Doc *openapi.OpenAPIDoc
}

type ServiceRecord struct {
	Service  Service
	Metadata ServiceMetadata
}

type MetadataMap map[Service]*ServiceMetadata

type inMemoryStore struct {
	mutex    sync.RWMutex
	limiter  limiter.Limiter
	climiter limiter.CountLimiter
	metadata MetadataMap
}

func newInMemoryStore(countLimiter limiter.CountLimiter, limiterImpl limiter.Limiter) *inMemoryStore {
	return &inMemoryStore{
		metadata: make(MetadataMap),
		limiter:  limiterImpl,
		climiter: countLimiter,
	}
}

func (s *inMemoryStore) Set(ks Service, m ServiceMetadata) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// If this is a new service... Increment the count.
	if _, ok := s.metadata[ks]; !ok {
		err := s.climiter.IncrementUsage(ks.Name)
		if err != nil {
			if s.limiter.IsHardLimitAtPointInTime() {
				return err
			} else {
				m.HasDoc = m.Doc != nil
				if m.HasDoc {
					m.Doc.Redact()
				}
			}
		}
	}

	s.metadata[ks] = &m
	return nil
}

func (s *inMemoryStore) Delete(ks Service) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.climiter.DecrementUsage(ks.Name)
	if err != nil {
		return err
	}

	delete(s.metadata, ks)
	return nil
}

func (s *inMemoryStore) Get(ks Service, with_doc bool) *ServiceMetadata {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	metadata := s.metadata[ks]
	if metadata == nil {
		return nil
	}
	result := &ServiceMetadata{
		Prefix:  metadata.Prefix,
		BaseURL: metadata.BaseURL,
		Doc:     metadata.Doc,
		HasDoc:  metadata.HasDoc,
	}
	if !with_doc {
		result.Doc = nil
	}
	return result
}

func (s *inMemoryStore) Slice() []ServiceRecord {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make([]ServiceRecord, 0)

	for service, metadata := range s.List() {
		result = append(result, ServiceRecord{
			Service: Service{
				Name:      service.Name,
				Namespace: service.Namespace,
			},
			Metadata: ServiceMetadata{
				Prefix:  metadata.Prefix,
				Doc:     metadata.Doc,
				HasDoc:  metadata.HasDoc,
				BaseURL: metadata.BaseURL,
			},
		})
	}

	sort.Slice(result, func(i, j int) bool {
		iFullName := fmt.Sprintf("%s.%s", result[i].Service.Namespace, result[i].Service.Name)
		jFullName := fmt.Sprintf("%s.%s", result[j].Service.Namespace, result[j].Service.Name)

		return iFullName < jFullName
	})

	return result
}

func (s *inMemoryStore) List() MetadataMap {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(MetadataMap)
	for service, metadata := range s.metadata {
		result[service] = &ServiceMetadata{
			Prefix:  metadata.Prefix,
			BaseURL: metadata.BaseURL,
			HasDoc:  metadata.HasDoc,
			Doc:     nil,
		}
	}
	return result
}
