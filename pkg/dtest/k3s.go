package dtest

import (
	"github.com/datawire/ambassador/pkg/dtest_k3s"
)

var DockerRegistry = dtest_k3s.DockerRegistry
var Kubeconfig = dtest_k3s.Kubeconfig
var GetKubeconfig = dtest_k3s.GetKubeconfig
var K3sUp = dtest_k3s.K3sUp
var K3sDown = dtest_k3s.K3sDown
