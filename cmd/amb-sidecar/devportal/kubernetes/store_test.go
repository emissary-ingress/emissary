package kubernetes

import (
	"testing"

	"github.com/Jeffail/gabs"
	. "github.com/onsi/gomega"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/openapi"
)

func testStoreInterface(s ServiceStore, t *testing.T) {
	g := NewGomegaWithT(t)

	sv1 := Service{Name: "a", Namespace: "b"}
	sv2 := Service{Name: "c", Namespace: "d"}
	json, _ := gabs.ParseJSON([]byte(`{"random":"json"}`))
	m1 := ServiceMetadata{
		Prefix: "x", BaseURL: "http://y", HasDoc: true,
		Doc: &openapi.OpenAPIDoc{JSON: json},
	}
	m1_nodoc := m1
	m1_nodoc.Doc = nil
	m2 := ServiceMetadata{
		Prefix: "x", BaseURL: "http://y", HasDoc: false,
		Doc: nil,
	}

	// Starts empty:
	start := s.List()
	if len(start) > 0 {
		t.Errorf("Store should start empty")
	}
	sv1_got := s.Get(sv1, false)
	if sv1_got != nil {
		t.Errorf("Got unexpected result")
	}

	// Can add an item with doc, and get it with and without doc:
	s.Set(sv1, m1)
	sv1_got = s.Get(sv1, true)
	g.Expect(&m1).To(Equal(sv1_got))
	sv1_got = s.Get(sv1, false)
	g.Expect(&m1_nodoc).To(Equal(sv1_got))

	// Can add another item without doc and list both:
	s.Set(sv2, m2)
	sv2_got := s.Get(sv2, true)
	g.Expect(&m2).To(Equal(sv2_got))
	expectedMap := make(MetadataMap)
	expectedMap[sv1] = &m1_nodoc
	expectedMap[sv2] = &m2
	g.Expect(s.List()).To(Equal(expectedMap))

	// Can delete an item:
	s.Delete(sv1)
	delete(expectedMap, sv1)
	g.Expect(s.List()).To(Equal(expectedMap))
	g.Expect(s.Get(sv1, false)).To(BeNil())

	// Delete final item:
	s.Delete(sv2)
	start = s.List()
	if len(start) > 0 {
		t.Errorf("Store should be empty")
	}
}

func TestStore(t *testing.T) {
	testStoreInterface(NewInMemoryStore(), t)
}
