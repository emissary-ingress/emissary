package agent_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/datawire/ambassador/pkg/agent"
	"github.com/datawire/ambassador/pkg/kates"
)

func sortPods(pods []*kates.Pod) {
	sort.SliceStable(pods, func(i, j int) bool {
		key1 := fmt.Sprintf("%s.%s", pods[i].ObjectMeta.Name, pods[i].ObjectMeta.Namespace)
		key2 := fmt.Sprintf("%s.%s", pods[j].ObjectMeta.Name, pods[j].ObjectMeta.Namespace)

		return key1 < key2
	})
}

func newPodPtr(name string, namespace string, labels map[string]string) *kates.Pod {
	pod := newPod(name, namespace, labels)
	return &pod
}

func newPod(name string, namespace string, labels map[string]string) kates.Pod {
	return kates.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func newService(name string, matchLabels map[string]string) *kates.Service {
	return &kates.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "name",
		},
		Spec: kates.ServiceSpec{
			Selector: matchLabels,
		},
	}
}

func TestPodStore(t *testing.T) {
	testcases := []struct {
		testName         string
		inputPods        []kates.Pod
		snapshotServices []*kates.Service
		expectedPods     []*kates.Pod
	}{
		{
			testName: "empty-snapshot",
			inputPods: []kates.Pod{
				newPod("pod1", "ns1", map[string]string{"label1": "label2"}),
			},
			snapshotServices: []*kates.Service{},
			expectedPods:     []*kates.Pod{},
		},
		{
			testName: "only-matching-pods",
			inputPods: []kates.Pod{
				newPod("pod1", "ns1", map[string]string{"label1": "matchvalue"}),
				newPod("pod2", "ns2", map[string]string{"label1": "matchvalue", "label2": "matchvalue", "extra": "label"}),
				newPod("pod3", "ns3", map[string]string{"label1": "matchvalue", "label2": "notmatchvalue"}),
				newPod("singlelabelmatch", "ns3", map[string]string{"single": "matchvalue", "tag": "notimportant"}),
				newPod("nopodlabels", "ns3", map[string]string{}),
			},
			snapshotServices: []*kates.Service{
				newService("myservice", map[string]string{"label1": "matchvalue", "label2": "matchvalue"}),
				newService("nomatchingpods", map[string]string{"label1": "matchvalue", "label2": "notapodvalue"}),
				newService("noselector", map[string]string{}),
				newService("singlelabelmatch", map[string]string{"single": "matchvalue"}),
			},
			expectedPods: []*kates.Pod{
				newPodPtr("singlelabelmatch", "ns3", map[string]string{"single": "matchvalue", "tag": "notimportant"}),
				newPodPtr("pod2", "ns2", map[string]string{"label1": "matchvalue", "label2": "matchvalue", "extra": "label"}),
			},
		},
		{
			testName: "nolabels",
			inputPods: []kates.Pod{
				newPod("nopodlabels", "ns3", map[string]string{}),
			},
			snapshotServices: []*kates.Service{
				newService("noselector", map[string]string{}),
			},
			expectedPods: []*kates.Pod{},
		},
		{
			testName:  "nopods",
			inputPods: []kates.Pod{},
			snapshotServices: []*kates.Service{
				newService("aservice", map[string]string{"hi": "whocares"}),
			},
			expectedPods: []*kates.Pod{},
		},
		{
			testName: "filtersimilarservices",
			inputPods: []kates.Pod{
				newPod("pod1", "ns", map[string]string{"product": "aes", "shooby": "doobie"}),
				newPod("pod2", "ns", map[string]string{"product": "aes", "shooby": "doobie"}),
				newPod("pod3", "ns", map[string]string{"product": "aes", "shooby": "boo"}),
			},
			snapshotServices: []*kates.Service{
				newService("aservice", map[string]string{"product": "aes", "shooby": "doobie"}),
				newService("bservice", map[string]string{"product": "aes", "shooby": "boo"}),
			},
			expectedPods: []*kates.Pod{
				newPodPtr("pod1", "ns", map[string]string{"product": "aes", "shooby": "doobie"}),
				newPodPtr("pod2", "ns", map[string]string{"product": "aes", "shooby": "doobie"}),
				newPodPtr("pod3", "ns", map[string]string{"product": "aes", "shooby": "boo"}),
			},
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.testName, func(innerT *testing.T) {
			ps := agent.NewPodStore(testcase.inputPods)

			assert.NotNil(innerT, ps)

			matchingPods := ps.GetPodsForServices(testcase.snapshotServices)

			assert.NotNil(innerT, matchingPods)
			assert.Equal(innerT, len(matchingPods), len(testcase.expectedPods))
			sortPods(matchingPods)
			sortPods(testcase.expectedPods)
			assert.Equal(innerT, testcase.expectedPods, matchingPods)
		})
	}

}
