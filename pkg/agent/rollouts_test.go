package agent

import (
	context "context"
	alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned/typed/rollouts/v1alpha1"
	"github.com/datawire/dlib/dlog"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"testing"
)

type mockRolloutsGetter struct {
	mockRolloutInterface *mockRolloutInterface
	latestNamespace      string
}

var _ v1alpha1.RolloutsGetter = &mockRolloutsGetter{}

func (m *mockRolloutsGetter) Rollouts(namespace string) v1alpha1.RolloutInterface {
	m.latestNamespace = namespace
	return m.mockRolloutInterface
}

type mockRolloutInterface struct {
	latestName      string
	latestPatchType types.PatchType
	latestOptions   metav1.PatchOptions
	patches         []string
	subresources    []string
}

var _ v1alpha1.RolloutInterface = &mockRolloutInterface{}

func (m *mockRolloutInterface) Create(ctx context.Context, rollout *alpha1.Rollout, opts metav1.CreateOptions) (*alpha1.Rollout, error) {
	panic("implement me")
}

func (m *mockRolloutInterface) Update(ctx context.Context, rollout *alpha1.Rollout, opts metav1.UpdateOptions) (*alpha1.Rollout, error) {
	panic("implement me")
}

func (m *mockRolloutInterface) UpdateStatus(ctx context.Context, rollout *alpha1.Rollout, opts metav1.UpdateOptions) (*alpha1.Rollout, error) {
	panic("implement me")
}

func (m *mockRolloutInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement me")
}

func (m *mockRolloutInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement me")
}

func (m *mockRolloutInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*alpha1.Rollout, error) {
	panic("implement me")
}

func (m *mockRolloutInterface) List(ctx context.Context, opts metav1.ListOptions) (*alpha1.RolloutList, error) {
	panic("implement me")
}

func (m *mockRolloutInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (m *mockRolloutInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *alpha1.Rollout, err error) {
	m.latestName = name
	m.latestPatchType = pt
	m.patches = append(m.patches, string(data))
	m.subresources = append(m.subresources, subresources...)
	m.latestOptions = opts
	return nil, nil
}

func TestRolloutCommand_RunWithClient(t *testing.T) {
	type fields struct {
		namespace   string
		rolloutName string
		action      rolloutAction
	}
	tests := []struct {
		name             string
		fields           fields
		wantPatches      []string
		wantSubresources []string
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "Pausing a rollout",
			fields: fields{
				namespace:   "default",
				rolloutName: "my-rollout",
				action:      rolloutActionPause,
			},
			wantPatches: []string{`{"spec":{"paused":true}}`},
			wantErr:     nil,
		},
		{
			name: "Aborting a rollout",
			fields: fields{
				namespace:   "default",
				rolloutName: "my-rollout",
				action:      rolloutActionAbort,
			},
			wantPatches:      []string{`{"status":{"abort":true}}`},
			wantSubresources: []string{"status"},
			wantErr:          nil,
		},
		{
			name: "Resume a rollout",
			fields: fields{
				namespace:   "default",
				rolloutName: "my-rollout",
				action:      rolloutActionResume,
			},
			wantPatches: []string{`{"spec":{"paused":false}}`, `{"status":{"abort":false}}`},
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRolloutInterface := &mockRolloutInterface{}
			mockRolloutsGetter := &mockRolloutsGetter{mockRolloutInterface: mockRolloutInterface}

			mockRolloutsFactory := rolloutsGetterFactory(func() (v1alpha1.RolloutsGetter, error) {
				return mockRolloutsGetter, nil
			})

			r := &rolloutCommand{
				namespace:   tt.fields.namespace,
				rolloutName: tt.fields.rolloutName,
				action:      tt.fields.action,
			}
			ctx := dlog.NewTestContext(t, true)
			err := r.RunWithClientFactory(ctx, mockRolloutsFactory)

			assert.NoError(t, err)
			assert.Equal(t, tt.fields.namespace, mockRolloutsGetter.latestNamespace)
			assert.Equal(t, tt.fields.rolloutName, mockRolloutInterface.latestName)
			assert.Equal(t, types.MergePatchType, mockRolloutInterface.latestPatchType)
			assert.Equal(t, tt.wantPatches, mockRolloutInterface.patches)
			assert.Equal(t, metav1.PatchOptions{}, mockRolloutInterface.latestOptions)
		})
	}
}
