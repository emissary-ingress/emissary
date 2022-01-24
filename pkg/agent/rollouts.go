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

// rolloutAction indicates the action to be performed on a Rollout object
type rolloutAction string

const (
	// rolloutActionPause represents the "pause" action on a Rollout
	rolloutActionPause = rolloutAction("pause")
	// rolloutActionResume represents the "resume" action on a Rollout
	rolloutActionResume = rolloutAction("resume")
	// rolloutActionAbort represents the "abort" action on a Rollout
	rolloutActionAbort = rolloutAction("abort")
)

// rolloutsGetterFactory is a factory for creating RolloutsGetter.
type rolloutsGetterFactory func() (argov1alpha1.RolloutsGetter, error)

// rolloutCommand holds a reference to a Rollout command to be ran.
type rolloutCommand struct {
	namespace   string
	rolloutName string
	action      rolloutAction
}

func (r *rolloutCommand) String() string {
	return fmt.Sprintf("<rollout=%s namespace=%s action=%s>", r.rolloutName, r.namespace, r.action)
}

// RunWithClientFactory runs the given Rollout command using rolloutsClientFactory to get a RolloutsGetter.
func (r *rolloutCommand) RunWithClientFactory(rolloutsClientFactory rolloutsGetterFactory) error {
	client, err := rolloutsClientFactory()
	if err != nil {
		return err
	}
	return r.patchRollout(client)
}

func (r *rolloutCommand) patchRollout(client argov1alpha1.RolloutsGetter) error {
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

// NewArgoRolloutsGetter creates a RolloutsGetter from Argo's v1alpha1 API.
func NewArgoRolloutsGetter() (argov1alpha1.RolloutsGetter, error) {
	kubeConfig, err := newK8sRestClient()
	if err != nil {
		return nil, err
	}

	argoClient, err := argov1alpha1.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return argoClient, nil
}

func newK8sRestClient() (*rest.Config, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}
