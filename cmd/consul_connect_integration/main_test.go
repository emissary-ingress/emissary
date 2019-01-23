package main

import (
	"strings"
	"testing"
)
import "github.com/datawire/apro/lib/testutil"

func TestNewAgent_AmbassadorIDToConsulServiceName(t *testing.T) {
	assert := testutil.Assert{T: t}

	tables := []struct {
		actual   string
		expected string
	}{
		{"", "ambassador"},
		{"ambassador", "ambassador-ambassador"},
		{"foo-bar-team", "ambassador-foo-bar-team"},
	}

	for _, table := range tables {
		a := NewAgent(table.actual, "UNUSED", "UNUSED", nil)
		assert.StrEQ(table.expected, a.ConsulServiceName)
	}
}

func TestNewAgent_SecretName(t *testing.T) {
	assert := testutil.Assert{T: t}

	tables := []struct {
		ambassadorID string
		secretName   string
		expected     string
	}{
		{"", "", "ambassador-consul-connect"},
		{"foobar", "", "ambassador-foobar-consul-connect"},
		{"foobar", "bazbot", "bazbot"},
	}

	for _, table := range tables {
		a := NewAgent(table.ambassadorID, "NAMESPACE", table.secretName, nil)
		assert.StrEQ(table.expected, a.SecretName)
	}
}

func TestFormatKubernetesSecretYAML(t *testing.T) {
	assert := testutil.Assert{T: t}

	certificate := "Ceci n'est pas une certificate"
	privateKey := "Ceci n'est pas une key"

	expected := `---
kind: Secret
apiVersion: v1
metadata:
    name: "IAmTheWalrus"
type: "kubernetes.io/tls"
data:
    tls.crt: "Q2VjaSBuJ2VzdCBwYXMgdW5lIGNlcnRpZmljYXRl"
    tls.key: "Q2VjaSBuJ2VzdCBwYXMgdW5lIGtleQ=="
`

	formatted := formatKubernetesSecretYAML("IAmTheWalrus", strings.Replace(certificate, "\n", "", -1), strings.Replace(privateKey, "\n", "", -1))
	assert.StrEQ(expected, formatted)
}

func TestCreateCertificateChain(t *testing.T) {
	assert := testutil.Assert{T: t}

	tables := []struct {
		root         string
		leaf         string
		intermediate []string
		expected     string
	}{
		{"ROOT\n", "LEAF\n", []string{}, `LEAF
ROOT
`},
		{"ROOT\n", "LEAF\n", []string{"INT0\n", "INT1\n", "INT2\n"}, `LEAF
INT0
INT1
INT2
ROOT
`},
	}

	for _, table := range tables {
		chain := createCertificateChain(table.root, table.leaf, table.intermediate)
		assert.StrEQ(table.expected, chain)
	}
}
