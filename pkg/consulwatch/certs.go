package consulwatch

import (
	"bytes"
	"encoding/base64"
	"os/exec"
	"strings"
	"text/template"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/supervisor"
)

const (
	// manifest that will be applied when the Consul certificate are obtained
	manifestTemplate = `
---
apiVersion: v1
kind: Secret
metadata:
  name: "{{.secret_name}}"
type: "kubernetes.io/tls"
data:
  tls.crt: "{{.tls_crt}}"
  tls.key: "{{.tls_key}}"
---
apiVersion: getambassador.io/v1
kind: TLSContext
metadata:
  name: ambassador-consul
spec:
  hosts: []
  secret: "{{.secret_name}}"
`
)

// replaceInTemplate performs replacements in an input text
func replaceInTemplate(text string, replacements map[string]interface{}) (string, error) {
	tmpl, err := template.New("template").Parse(text)
	if err != nil {
		return "", err
	}

	b := bytes.Buffer{}
	if err := tmpl.Execute(&b, replacements); err != nil {
		return "", err
	}
	return b.String(), nil
}

// certChain is a certificates chain that will be translated to a k8s certificate
type certChain struct {
	process *supervisor.Process
	CA      *CARoots
	Leaf    *Certificate
}

func newCertChain(process *supervisor.Process) certChain {
	return certChain{
		process: process,
	}
}

// WriteTo writes the certificates chain to a Kubernetes Secret
func (c *certChain) WriteTo(secretNamespace, secretName string) error {
	if c.CA == nil || c.Leaf == nil {
		return nil // we need both CA & Leaf for creating/updating the secret
	}

	temp := c.CA.Roots[c.CA.ActiveRootID]
	caRoot := &temp

	// create the certificate chain
	chain := c.Leaf.PEM + caRoot.PEM

	// format the kubernetes secret
	chain64 := base64.StdEncoding.EncodeToString([]byte(chain))
	key64 := base64.StdEncoding.EncodeToString([]byte(c.Leaf.PrivateKeyPEM))
	replacements := map[string]interface{}{
		"secret_name": secretName,
		"tls_crt":     chain64,
		"tls_key":     key64,
	}
	secret, err := replaceInTemplate(manifestTemplate, replacements)
	if err != nil {
		return err
	}

	logger.Printf("Creating/updating TLS certificate secret: %s/%s", secretNamespace, secretName)
	kubeinfo := k8s.NewKubeInfo("", "", secretNamespace)
	args, err := kubeinfo.GetKubectlArray("apply", "-f", "-")
	if err != nil {
		return err
	}
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}
	apply := c.process.Command(kubectl, args...)
	apply.Stdin = strings.NewReader(secret)
	err = apply.Start()
	if err != nil {
		return err
	}
	err = c.process.DoClean(apply.Wait, apply.Process.Kill)
	if err != nil {
		return err
	}
	logger.Printf("certificate secret: %s/%s", secretNamespace, secretName)

	return nil
}
