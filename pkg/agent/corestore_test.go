package agent_test

import (
	"testing"

	"github.com/emissary-ingress/emissary/v3/pkg/agent"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCoreStore(t *testing.T) {
	type testCases struct {
		name                string
		getPods             func() []*kates.Pod
		expectedPods        int
		getConfigMaps       func() []*kates.ConfigMap
		expectedConfigMaps  int
		getDeployments      func() []*kates.Deployment
		expectedDeployments int
		getEndpoints        func() []*kates.Endpoints
		expectedEndpoints   int
	}
	cases := []*testCases{
		{
			name: "will add running endpoints to state of the world",
			getEndpoints: func() []*kates.Endpoints {
				return []*kates.Endpoints{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Endpoints",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-endpoint",
							Namespace: "default",
						},
					},
				}
			},
			expectedEndpoints: 1,
		},
		{
			name: "will add running pods to state of the world",
			getPods: func() []*kates.Pod {
				return []*kates.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-pod",
							Namespace: "default",
						},
						Status: v1.PodStatus{
							Phase: v1.PodRunning,
						},
					},
				}
			},
			expectedPods: 1,
		},
		{
			name: "will ensure no duplicate pods are added",
			getPods: func() []*kates.Pod {
				return []*kates.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-pod",
							Namespace: "default",
						},
						Status: v1.PodStatus{
							Phase: v1.PodRunning,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-pod",
							Namespace: "default",
						},
						Status: v1.PodStatus{
							Phase: v1.PodRunning,
						},
					},
				}
			},
			expectedPods: 1,
		},
		{
			name: "will exclude kube-system pods from state of the world",
			getPods: func() []*kates.Pod {
				return []*kates.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-pod",
							Namespace: "kube-system",
						},
						Status: v1.PodStatus{
							Phase: v1.PodRunning,
						},
					},
				}
			},
			expectedPods: 0,
		},
		{
			name: "will exclude non running pods from state of the world",
			getPods: func() []*kates.Pod {
				return []*kates.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-pod",
							Namespace: "defautl",
						},
						Status: v1.PodStatus{
							Phase: v1.PodSucceeded,
						},
					},
				}
			},
			expectedPods: 0,
		},
		{
			name: "will send pods in failed state",
			getPods: func() []*kates.Pod {
				return []*kates.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-pod",
							Namespace: "defautl",
						},
						Status: v1.PodStatus{
							Phase: v1.PodFailed,
						},
					},
				}
			},
			expectedPods: 1,
		},
		{
			name: "will add configmaps to the configmapStore successfully",
			getConfigMaps: func() []*kates.ConfigMap {
				return []*kates.ConfigMap{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ConfigMap",
							APIVersion: "",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-config-map",
							Namespace: "default",
						},
					},
				}
			},
			expectedConfigMaps: 1,
		},
		{
			name: "will ensure no duplicated configmaps are added",
			getConfigMaps: func() []*kates.ConfigMap {
				return []*kates.ConfigMap{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ConfigMap",
							APIVersion: "",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-config-map",
							Namespace: "default",
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ConfigMap",
							APIVersion: "",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-config-map",
							Namespace: "default",
						},
					},
				}
			},
			expectedConfigMaps: 1,
		},
		{
			name: "will exclude configmaps from kube-system",
			getConfigMaps: func() []*kates.ConfigMap {
				return []*kates.ConfigMap{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ConfigMap",
							APIVersion: "",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-config-map",
							Namespace: "kube-system",
						},
					},
				}
			},
			expectedConfigMaps: 0,
		},
		{
			name: "will add Deployments to state of the world",
			getDeployments: func() []*kates.Deployment {
				return []*kates.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-deployment",
							Namespace: "default",
						},
					},
				}
			},
			expectedDeployments: 1,
		},
		{
			name: "will ensure no duplicated deployments are added",
			getDeployments: func() []*kates.Deployment {
				return []*kates.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-deployment",
							Namespace: "default",
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-deployment",
							Namespace: "default",
						},
					},
				}
			},
			expectedDeployments: 1,
		},
		{
			name: "will exclude kube-system Deployments from state of the world",
			getDeployments: func() []*kates.Deployment {
				return []*kates.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-deployment",
							Namespace: "kube-system",
						},
					},
				}
			},
			expectedDeployments: 0,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.getEndpoints != nil {
				endpoints := c.getEndpoints()
				endpointStore := agent.NewEndpointsStore(endpoints)
				epSOW := endpointStore.StateOfWorld()
				assert.Equal(t, c.expectedEndpoints, len(epSOW))
			}
			if c.getPods != nil {
				pods := c.getPods()
				podStore := agent.NewPodStore(pods)
				podSOTW := podStore.StateOfWorld()
				if c.expectedPods != len(podSOTW) {
					t.Errorf("error: expected %d pods but found %d", c.expectedPods, len(podSOTW))
				}
			}
			if c.getConfigMaps != nil {
				cms := c.getConfigMaps()
				cmStore := agent.NewConfigMapStore(cms)
				cmSOTW := cmStore.StateOfWorld()
				if c.expectedConfigMaps != len(cmSOTW) {
					t.Errorf("error: expected %d configmaps found %d", c.expectedConfigMaps, len(cmSOTW))
				}
			}
			if c.getDeployments != nil {
				dep := c.getDeployments()
				depStore := agent.NewDeploymentStore(dep)
				depSOTW := depStore.StateOfWorld()
				if c.expectedDeployments != len(depSOTW) {
					t.Errorf("error: expected %d deployments found %d", c.expectedDeployments, len(depSOTW))
				}
			}
		})
	}
}
