package k8scerts

import (
	_ "embed"
)

//go:embed cert.pem
var K8sCert string

//go:embed cert.key
var K8sKey string
