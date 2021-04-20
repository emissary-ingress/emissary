package agent

import (
	"fmt"

	"github.com/datawire/ambassador/pkg/kates"
)

// Store pods so we can easily filter them based on the selectors defined in the services
type podStore struct {
	podKeyByLabels podKeyByLabels
	// key the pods on podname.podnamespace so we can return the actual pod obj
	podsByKey map[string]kates.Pod
}

type podKeyByLabels struct {
	// okay yeah i know this is a lot, but it makes finding the pods we care about a lot easier
	// this struct is map[PodLabelName]map[PodLabelValue]map[PodName.PodNamespace]bool
	// so if a pod with name mypod and namespace myns had the labels [label1=value1,label2=value2],
	// you could find it in this map at podKeyByLabels[label1][value1][mypod.namespace] and
	// podKeyByLabels[label2][value2][mypod.namespace]
	labelToPodKeyMap map[string]map[string]map[string]bool
}

func (pl *podKeyByLabels) addPodsWithLabel(podKey string, labelKey string, labelValue string) {
	if pl.labelToPodKeyMap == nil {
		pl.labelToPodKeyMap = map[string]map[string]map[string]bool{}
	}
	if _, ok := pl.labelToPodKeyMap[labelKey]; !ok {
		pl.labelToPodKeyMap[labelKey] = map[string]map[string]bool{}
	}
	if _, ok := pl.labelToPodKeyMap[labelKey][labelValue]; !ok {
		pl.labelToPodKeyMap[labelKey][labelValue] = map[string]bool{}
	}
	pl.labelToPodKeyMap[labelKey][labelValue][podKey] = true
}

func (pl *podKeyByLabels) getPodKeysWithLabel(labelKey string, labelValue string) map[string]bool {
	if _, exists := pl.labelToPodKeyMap[labelKey]; !exists {
		// no pods have the label, move right along...
		return map[string]bool{}
	}
	currPodsWithLabel, exists := pl.labelToPodKeyMap[labelKey][labelValue]
	if !exists {
		// no pods have label with value, move right along...
		return map[string]bool{}
	}
	ret := map[string]bool{}
	for k, v := range currPodsWithLabel {
		ret[k] = v
	}
	return ret
}

func NewPodStore(pods []kates.Pod) *podStore {
	ps := &podStore{
		podKeyByLabels: podKeyByLabels{},
		podsByKey:      map[string]kates.Pod{},
	}

	for _, pod := range pods {
		podKey := fmt.Sprintf("%s.%s", pod.GetName(), pod.GetNamespace())
		ps.podsByKey[podKey] = pod
		for labelKey, labelValue := range pod.ObjectMeta.Labels {
			ps.podKeyByLabels.addPodsWithLabel(podKey, labelKey, labelValue)
		}
	}
	return ps
}

func (ps *podStore) GetPodsForServices(services []*kates.Service) []*kates.Pod {
	podNames := map[string]bool{}
OUTER:
	for _, svc := range services {
		podsMatchingSvc := map[string]bool{}
		first := true
		// for services that don't have selectors, (i.e. ExternalName typed svcs), this will
		// the selector map will be empty, so don't bother checking the service type. the
		// right thing will happen
		for labelKey, labelValue := range svc.Spec.Selector {

			// get all the pods with label labelKey=labelValue
			currPodsWithLabel := ps.podKeyByLabels.getPodKeysWithLabel(labelKey, labelValue)
			if len(currPodsWithLabel) == 0 {
				// if there are no pods that have labelKey=labelValue, no pods match
				// this service's selectors, so let's move to the next service
				continue OUTER
			}
			if first {
				// if this is the first label, all the pods with it are still valid
				// matches for the service
				first = false
				podsMatchingSvc = currPodsWithLabel
				continue
			}
			for podKey := range podsMatchingSvc {
				// if a pod in the current pod list for the service does NOT have
				// the label we're currently examining, take it out of the matching
				// pod list.
				// i.e. if a pod only has labels [label1=value1] but the service is
				// selecting on label1=value1 AND label2=value2, we should remove
				// the pod from the list (because it does _not_ match the service)
				if _, exists := currPodsWithLabel[podKey]; !exists {
					delete(podsMatchingSvc, podKey)
				}
			}

		}
		for podName := range podsMatchingSvc {
			podNames[podName] = true
		}
	}
	// okay, now for all the pod keys we've collected, we'll construct a list of all the pods to
	// return
	ret := []*kates.Pod{}
	for podName := range podNames {
		pod, ok := ps.podsByKey[podName]
		if ok {
			ret = append(ret, &pod)
		}
	}
	return ret
}
