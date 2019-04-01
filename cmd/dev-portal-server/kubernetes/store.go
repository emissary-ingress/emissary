package kubernetes

import (
	"sync"
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
)

type Service struct {
	Name string
	Namespace string
}

type ServiceMetadata struct {
	Prefix string
	HasDoc bool
	// May be nil even if HasDoc is true if loaded without the doc.
	Doc *openapi.OpenAPIDoc
	// TODO: Extend with virtual hosts, and other routing options.
}

type MetadataMap map[Service]*ServiceMetadata

// Storage for metadata about Kubernetes services. Implementations should assume
// access from multiple goroutines.
type ServiceStore interface {
	// Store new metadata for a service. The OpenAPIDoc is presumed to
	// already have been appropriately updated, e.g. prefixes munged.
	set(ks Service, m ServiceMetadata)
	// Retrieve metadata or a service, optionally loading the OpenAPI doc if
	// there is one.
	get(ks Service, with_doc bool) *ServiceMetadata
	// Get all services' metadata. OpenAPI docs are not loaded.
	list() MetadataMap
}

// In-memory implementation of ServiceStore.
type inMemoryStore struct {
	mutex sync.RWMutex
	metadata MetadataMap
}

// Create in-memory implementation of ServiceStore.
func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{}
}

func (s *inMemoryStore) set(ks Service, m ServiceMetadata) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.metadata[ks] = &m
}

func (s *inMemoryStore) get(ks Service, with_doc bool) *ServiceMetadata {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	metadata := s.metadata[ks]
	if (metadata == nil) {
		return nil
	}
	result := &ServiceMetadata{
		Prefix: metadata.Prefix,
		Doc: metadata.Doc,
		HasDoc: metadata.HasDoc,
	}
	if (!with_doc) {
		result.Doc = nil
	}
	return result
}

func (s *inMemoryStore) list() MetadataMap {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(MetadataMap)
	for service, metadata := range s.metadata {
		result[service] = &ServiceMetadata{
			Prefix: metadata.Prefix,
			HasDoc: metadata.HasDoc,
			Doc: nil,
		}
	}
	return result
}
