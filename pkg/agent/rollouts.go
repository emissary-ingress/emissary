package agent

import (
	"context"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned/typed/rollouts/v1alpha1"
	"github.com/datawire/dlib/dlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type rolloutAction string

const (
	rolloutActionPause  = rolloutAction("pause")
	rolloutActionResume = rolloutAction("resume")
	rolloutActionAbort  = rolloutAction("abort")
)

type RolloutCommand struct {
	namespace   string
	rolloutName string
	action      rolloutAction
}

func (r *RolloutCommand) Run() {
	client, err := newRolloutsClient()
	if err != nil {
		return
	}
	r.patchRollout(client)
}

func (r *RolloutCommand) patchRollout(client *argov1alpha1.ArgoprojV1alpha1Client) {
	var patch []byte
	switch r.action {
	case rolloutActionResume:
		patch = []byte(`{"spec":{"paused":false}}`)
	case rolloutActionAbort:
		patch = []byte(`{"status":{"abort":true}}`)
	case rolloutActionPause:
		patch = []byte(`{"spec":{"paused":true}}`)
	}
	rollout := client.Rollouts(r.namespace)
	_, err := rollout.Patch(
		context.TODO(),
		r.rolloutName,
		types.MergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		panic(err)
	}
	dlog.Infof(context.TODO(), "rollout '%s' paused\n", r.rolloutName)
}

func newRolloutsClient() (*argov1alpha1.ArgoprojV1alpha1Client, error) {
	kubeConfig, err := newConfig()
	if err != nil {
		return nil, err
	}

	argoClient, err := argov1alpha1.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return argoClient, nil
}

func newConfig() (*rest.Config, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}
