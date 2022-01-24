package agent

import (
	"context"
	"fmt"

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

func (r *RolloutCommand) RunWithClient(client argov1alpha1.RolloutsGetter) error {
	return r.patchRollout(client)
}

func (r *RolloutCommand) RunWithDefaultClient() error {
	client, err := newRolloutsClient()
	if err != nil {
		return err
	}
	return r.patchRollout(client)
}

func (r *RolloutCommand) patchRollout(client argov1alpha1.RolloutsGetter) error {
	var patch []byte
	switch r.action {
	case rolloutActionResume:
		patch = []byte(`{"spec":{"paused":false}}`)
	case rolloutActionAbort:
		patch = []byte(`{"status":{"abort":true}}`)
	case rolloutActionPause:
		patch = []byte(`{"spec":{"paused":true}}`)
	default:
		err := fmt.Errorf(
			"tried to perform unknown action '%s' on rollout %s (%s)",
			r.action,
			r.rolloutName,
			r.namespace,
		)
		dlog.Errorln(context.TODO(), err)
		return err
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
		errMsg := fmt.Errorf(
			"failed to %s rollout %s (%s): %w",
			r.action,
			r.rolloutName,
			r.namespace,
			err,
		)
		dlog.Errorln(context.TODO(), errMsg)
		return err
	}
	return nil
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
