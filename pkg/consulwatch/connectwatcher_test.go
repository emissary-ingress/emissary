package consulwatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNamespaceAndName(t *testing.T) {

	tables := []struct {
		secret            string
		expectedNamespace string
		expectedName      string
	}{
		{"default/my-secret", "default", "my-secret"},
		{"my-secret", "", "my-secret"},
		{"", "", ""},
	}

	for _, table := range tables {
		namespace, name := getNamespaceAndName(table.secret)
		assert.Equal(t, table.expectedNamespace, namespace)
		assert.Equal(t, table.expectedName, name)
	}
}
