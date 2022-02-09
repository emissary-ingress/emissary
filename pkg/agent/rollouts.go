package agent

import (
	"context"
	"fmt"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned/typed/rollouts/v1alpha1"
	"github.com/datawire/dlib/dlog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// rolloutAction indicates the action to be performed on a Rollout object
type rolloutAction string

const (
	// rolloutActionPause represents the "pause" action on a Rollout
	rolloutActionPause = rolloutAction("PAUSE")
	// rolloutActionResume represents the "resume" action on a Rollout
	rolloutActionResume = rolloutAction("RESUME")
	// rolloutActionAbort represents the "abort" action on a Rollout
	rolloutActionAbort = rolloutAction("ABORT")
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
func (r *rolloutCommand) RunWithClientFactory(ctx context.Context, rolloutsClientFactory rolloutsGetterFactory) error {
	client, err := rolloutsClientFactory()
	if err != nil {
		return err
	}
	return r.patchRollout(ctx, client)
}

const unpausePatch = `{"spec":{"paused":false}}`
const abortPatch = `{"status":{"abort":true}}`
const retryPatch = `{"status":{"abort":false}}`
const pausePatch = `{"spec":{"paused":true}}`

func (r *rolloutCommand) patchRollout(ctx context.Context, client argov1alpha1.RolloutsGetter) error {
	var err error
	switch r.action {
	// The "Resume" action in the DCP should be able to recover from Rollout that is either paused or aborted.
	// For more information about the need for rolloutCommand.applyRetryPatch to apply the "retry" patch, please check its godoc.
	case rolloutActionResume:
		err = r.applyPatch(ctx, client, unpausePatch)
		if err == nil {
			err = r.applyRetryPatch(ctx, client)
		}
	case rolloutActionAbort:
		err = r.applyPatch(ctx, client, abortPatch)
	case rolloutActionPause:
		err = r.applyPatch(ctx, client, pausePatch)
	default:
		err := fmt.Errorf(
			"tried to perform unknown action '%s' on rollout %s (%s)",
			r.action,
			r.rolloutName,
			r.namespace,
		)
		dlog.Errorln(ctx, err)
		return err
	}
	if err != nil {
		errMsg := fmt.Errorf(
			"failed to %s rollout %s (%s): %w",
			r.action,
			r.rolloutName,
			r.namespace,
			err,
		)
		dlog.Errorln(ctx, errMsg)
		return err
	}
	return nil
}

func (r *rolloutCommand) applyPatch(ctx context.Context, client argov1alpha1.RolloutsGetter, patch string) error {
	rollout := client.Rollouts(r.namespace)
	_, err := rollout.Patch(
		ctx,
		r.rolloutName,
		types.MergePatchType,
		[]byte(patch),
		metav1.PatchOptions{},
	)
	return err
}

// applyRetryPatch exists because to "retry" a Rollout, recovering it from the "aborted" state, Argo Rollouts
// first tries to patch the rollouts/status subresource. If that fails, then base "rollouts" rollout is patched.
// This is based on the logic of the Argo Rollouts CLI, as seen at https://github.com/argoproj/argo-rollouts/blob/v1.1.1/pkg/kubectl-argo-rollouts/cmd/retry/retry.go#L84.
func (r *rolloutCommand) applyRetryPatch(ctx context.Context, client argov1alpha1.RolloutsGetter) error {
	rollout := client.Rollouts(r.namespace)
	_, err := rollout.Patch(
		ctx,
		r.rolloutName,
		types.MergePatchType,
		[]byte(retryPatch),
		metav1.PatchOptions{},
		"status",
	)
	if err != nil && k8serrors.IsNotFound(err) {
		_, err = rollout.Patch(
			ctx,
			r.rolloutName,
			types.MergePatchType,
			[]byte(retryPatch),
			metav1.PatchOptions{},
		)
	}
	return err
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
