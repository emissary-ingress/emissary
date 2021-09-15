package dtest

import (
	"github.com/datawire/ambassador/v2/pkg/dtest_k3s"
)

var DockerRegistry = dtest_k3s.DockerRegistry
var Kubeconfig = dtest_k3s.Kubeconfig
var GetKubeconfig = dtest_k3s.GetKubeconfig
var K3sUp = dtest_k3s.K3sUp
var K3sDown = dtest_k3s.K3sDown
var RegistryUp = dtest_k3s.RegistryUp
var RegistryDown = dtest_k3s.RegistryDown
var WithMachineLock = dtest_k3s.WithMachineLock
var WithNamedMachineLock = dtest_k3s.WithNamedMachineLock
