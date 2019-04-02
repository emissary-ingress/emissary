package kubernetes

import (
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
	"sync"
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

type MetadataMap map[Service]*ServiceMetadata

// Storage for metadata about Kubernetes services. Implementations should assume
// access from multiple goroutines.
type ServiceStore interface {
	// Store new metadata for a service. The OpenAPIDoc is presumed to
	// already have been appropriately updated, e.g. prefixes munged.
	Set(ks Service, m ServiceMetadata)
	// Retrieve metadata or a service, optionally loading the OpenAPI doc if
	// there is one.
	Get(ks Service, with_doc bool) *ServiceMetadata
	// Get all services' metadata. OpenAPI docs are not loaded.
	List() MetadataMap
}

// In-memory implementation of ServiceStore.
type inMemoryStore struct {
	mutex    sync.RWMutex
	metadata MetadataMap
}

// Create in-memory implementation of ServiceStore.
func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{metadata: make(MetadataMap)}
}

func (s *inMemoryStore) Set(ks Service, m ServiceMetadata) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.metadata[ks] = &m
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
