package consulwatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceInTemplate(t *testing.T) {
	tables := []struct {
		values   map[string]interface{}
		template string
		expected string
	}{
		{
			map[string]interface{}{
				"A": "first",
				"B": "second",
			},
			"{{.A}}-{{.B}}",
			"first-second"},
	}

	for _, table := range tables {
		res, err := replaceInTemplate(table.template, table.values)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, table.expected, res)
	}
}

func TestNewAgent_AmbassadorIDToConsulServiceName(t *testing.T) {
	tables := []struct {
		actual   string
		expected string
	}{
		{"", "ambassador"},
		{"ambassador", "ambassador-ambassador"},
		{"foo-bar-team", "ambassador-foo-bar-team"},
	}
	secret := "UNUSED/UNUSED"

	for _, table := range tables {
		a := NewAgent(ConsulWatchSpec{Id: table.actual, Secret: secret})
		assert.Equal(t, table.expected, a.ConsulServiceName)
	}
}

func TestNewAgent_SecretName(t *testing.T) {

	tables := []struct {
		ambassadorID string
		secretName   string
		expected     string
	}{
		{"", "", "ambassador-consul-connect"},
		{"foobar", "", "ambassador-foobar-consul-connect"},
		{"foobar", "NAMESPACE/bazbot", "bazbot"},
	}

	for _, table := range tables {
		a := NewAgent(ConsulWatchSpec{Id: table.ambassadorID, Secret: table.secretName})
		assert.Equal(t, table.expected, a.SecretName)
	}
}
