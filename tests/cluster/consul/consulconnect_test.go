// +build test

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/datawire/apro/lib/testutil"
	"github.com/hashicorp/consul/api"
)

func TestConsulConnectTLSCertificateChainIsPresentAsKubernetesSecret(t *testing.T) {
	assert := testutil.Assert{T: t}

	timeout := time.After(30 * time.Second)
	tick := time.Tick(1 * time.Second)

	config := api.DefaultConfig()
	config.Address = "consul-server:8500"

	consul, err := api.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

Loop:
	for {
		select {
		case <-timeout:
			t.FailNow()
		case <-tick:
			data, err := kubectlGetSecret("default", "ambassador-consul-connect")
			if err != nil {
				// try again because it is entirely possible the TLS certificate secret has not yet been created on
				// kubernetes
				continue
			}

			if data != "" {
				secret, err := gabs.ParseJSON([]byte(data))
				if err != nil {
					t.Fatal(err)
				}

				secretName := secret.Path("metadata.name").Data().(string)
				assert.StrEQ("ambassador-consul-connect", secretName)

				privateKeyBase64 := secret.Search("data", "tls.key")
				certificateChainBase64 := secret.Search("data", "tls.crt")

				b64 := base64.StdEncoding
				privateKeyBytes, err := b64.DecodeString(privateKeyBase64.Data().(string))
				if err != nil {
					t.Fatal(err)
				}

				certificateChainBytes, err := b64.DecodeString(certificateChainBase64.Data().(string))
				if err != nil {
					t.Fatal(err)
				}

				leaf, err := consulGetLeafCert(consul, "ambassador")
				if err != nil {
					t.Fatal(err)
				}

				assert.StrEQ(leaf.PrivateKeyPEM, string(privateKeyBytes))

				rootCA, err := consulGetRoot(consul)
				if err != nil {
					t.Fatal(err)
				}

				assert.StrEQ(string(certificateChainBytes), fmtCertificateChain(leaf.CertPEM, rootCA.RootCertPEM))

				break Loop
			}
		}
	}
}

func TestConsulConnectTLSCertificateChainIsUpdatedWhenConnectRootCAChanges(t *testing.T) {
	var err error

	assert := testutil.Assert{T: t}

	timeout := time.After(30 * time.Second)
	tick := time.Tick(1 * time.Second)

	config := api.DefaultConfig()
	config.Address = "consul-server:8500"

	consul, err := api.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	initialSecret := ""
	updatedSecret := ""

Loop1:
	for {
		select {
		case <-timeout:
			t.FailNow()
		case <-tick:
			initialSecret, err = kubectlGetSecret("default", "ambassador-consul-connect")
			if err != nil {
				// try again because it is entirely possible the TLS certificate secret has not yet been created on
				// kubernetes
				continue
			}

			if initialSecret != "" {
				break Loop1
			}
		}
	}

	if err := consulKubeRotate("default"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

Loop2:
	for {
		select {
		case <-timeout:
			t.FailNow()
		case <-tick:
			updatedSecret, err = kubectlGetSecret("default", "ambassador-consul-connect")
			if err != nil {
				t.Fatal(err)
			}

			break Loop2
		}
	}

	assert.StrNotEQ(initialSecret, updatedSecret)

	secret, err := gabs.ParseJSON([]byte(updatedSecret))
	if err != nil {
		t.Fatal(err)
	}

	privateKeyBase64 := secret.Search("data", "tls.key")
	certificateChainBase64 := secret.Search("data", "tls.crt")

	b64 := base64.StdEncoding
	privateKeyBytes, err := b64.DecodeString(privateKeyBase64.Data().(string))
	if err != nil {
		t.Fatal(err)
	}

	certificateChainBytes, err := b64.DecodeString(certificateChainBase64.Data().(string))
	if err != nil {
		t.Fatal(err)
	}

	leaf, err := consulGetLeafCert(consul, "ambassador")
	if err != nil {
		t.Fatal(err)
	}

	assert.StrEQ(leaf.PrivateKeyPEM, string(privateKeyBytes))

	rootCA, err := consulGetRoot(consul)
	if err != nil {
		t.Fatal(err)
	}

	assert.StrEQ(string(certificateChainBytes), fmtCertificateChain(leaf.CertPEM, rootCA.RootCertPEM))

}

func fmtCertificateChain(leafCertificate string, rootCerts string) string {
	chain := []string{leafCertificate, rootCerts}
	return strings.Join(chain, "")
}

func consulGetRoot(consul *api.Client) (*api.CARoot, error) {
	var root = &api.CARoot{}
	var err error

	rootList, _, err := consul.Agent().ConnectCARoots(&api.QueryOptions{})
	if err != nil {
		return nil, err
	}

	for _, r := range rootList.Roots {
		if r.Active && r.ID == rootList.ActiveRootID {
			root = r
			break
		}
	}

	if root == nil {
		err = errors.New("ca root not found")
	}

	return root, err
}

func consulGetLeafCert(consul *api.Client, service string) (*api.LeafCert, error) {
	leaf, _, err := consul.Agent().ConnectCALeaf(service, &api.QueryOptions{})
	return leaf, err
}

func consulKubeRotate(namespace string) error {
	consulKube, err := exec.LookPath("consul-kube")
	if err != nil {
		return err
	}

	args := []string{"-namespace=" + namespace, "rotate"}
	cmd := exec.Command(consulKube, args...)

	out, err := cmd.Output()
	fmt.Println(out)
	return err
}

func kubectlGetSecret(namespace string, name string) (string, error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", err
	}

	namespaceArg := make([]string, 0)
	if namespace == "" {
		namespaceArg = append(namespaceArg, "--namespace="+namespace)
	}

	args := []string{"get", "secret", name, "--output=json", "--ignore-not-found"}
	args = append(args, namespaceArg...)
	cmd := exec.Command(kubectl, args...)

	out, err := cmd.Output()
	return string(out), err
}
