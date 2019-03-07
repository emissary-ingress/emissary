package consul

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/Jeffail/gabs"
	"github.com/datawire/apro/lib/testutil"
	"github.com/hashicorp/consul/api"
	"math/big"
	"os/exec"
	"strings"
	"testing"
	"time"
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
	config := api.DefaultConfig()
	config.Address = "consul-server:8500"

	consul, err := api.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	err = consulRotateRootCA(consul)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsulConnectTLSCertificateChainIsUpdatedWhenLeafCertificateChanges(t *testing.T) {

}

//func rotateConsulRootCACertificate() {
//
//}

func fmtCertificateChain(leafCertificate string, rootCerts string) string {
	chain := []string{leafCertificate, rootCerts}
	return strings.Join(chain, "")
}

func generateECDSAPrivateKey() (*ecdsa.PrivateKey, error) {
	publicKeyCurve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(publicKeyCurve, rand.Reader)
	return privateKey, err
}

func generateCARoot(serial int64, trustDomain string, publicKey *ecdsa.PublicKey) (*x509.Certificate, error) {
	spiffeURI := fmt.Sprintf("URI:spiffe://%s", trustDomain)

	publicKeyPEM, _ := x509.MarshalPKIXPublicKey(publicKey)

	ca := &x509.Certificate{
		Version:      2,
		SerialNumber: big.NewInt(serial),
		PublicKey:    publicKey,
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(10*365*86400, 0),
		Subject:      pkix.Name{CommonName: fmt.Sprintf("Consul CA %d", serial)},
		Extensions: []pkix.Extension{
			// asn1 oid "Id" = subjectKeyIdentifier
			{Id: []int{2, 5, 29, 14}, Critical: false, Value: []byte("hash")},

			// asn1 oid "Id" = authorityKeyIdentifier
			{Id: []int{2, 5, 29, 35}, Critical: false, Value: []byte("keyid:always,issuer")},

			// asn1 oid "Id" = keyUsage
			{Id: []int{2, 5, 29, 15}, Critical: false, Value: []byte("digitalSignature, cRLSign, keyCertSign")},

			// asn1 oid "Id" = subjectAltName
			{Id: []int{2, 5, 29, 17}, Critical: false, Value: []byte(spiffeURI)},

			// asn1 oid "Id" = basicConstraints
			{Id: []int{2, 5, 29, 19}, Critical: true, Value: []byte("CA:TRUE")},
		},
		Issuer:                pkix.Name{CommonName: fmt.Sprintf("Consul CA %d", serial)},
		IsCA:                  true,
		BasicConstraintsValid: true,

		Signature:          publicKeyPEM,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}

	return ca, nil
}

func encodePEM(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string) {
	x509Encoded, _ := x509.MarshalECPrivateKey(privateKey)
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	x509EncodedPub, _ := x509.MarshalPKIXPublicKey(publicKey)
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncoded), string(pemEncodedPub)
}

func consulRotateRootCA(consul *api.Client) error {
	privateKey, err := generateECDSAPrivateKey()
	if err != nil {
		return err
	}

	privateKeyPEM, publicKeyPEM := encodePEM(privateKey, &privateKey.PublicKey)
	fmt.Println(string(privateKeyPEM))
	fmt.Println(string(publicKeyPEM))

	//currentRoot, err := consulGetRoot(consul)
	//if err != nil {
	//	return err
	//}
	//
	return nil
}

func consulGetRoot(consul *api.Client) (*api.CARoot, error) {
	var root = &api.CARoot{}
	var err error

	rootList, _, err := consul.Agent().ConnectCARoots(&api.QueryOptions{})
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
	var leaf = &api.LeafCert{}
	var err error

	leaf, _, err = consul.Agent().ConnectCALeaf(service, &api.QueryOptions{})
	return leaf, err
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
