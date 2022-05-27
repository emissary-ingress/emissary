package dtest

import (
	dtest_k3s "github.com/datawire/dtest"
)

type KubeVersion = dtest_k3s.KubeVersion

var Kube22 = dtest_k3s.Kube22

var DockerRegistry = dtest_k3s.DockerRegistry
var KubeVersionConfig = dtest_k3s.KubeVersionConfig
var GetKubeconfig = dtest_k3s.GetKubeconfig
var K3sVersionUp = dtest_k3s.K3sVersionUp
var K3sDown = dtest_k3s.K3sDown
var RegistryUp = dtest_k3s.RegistryUp
var RegistryDown = dtest_k3s.RegistryDown
var WithMachineLock = dtest_k3s.WithMachineLock
var WithNamedMachineLock = dtest_k3s.WithNamedMachineLock
