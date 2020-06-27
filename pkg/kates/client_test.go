package kates

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCRUD(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-upsert-configmap",
		},
	}

	assert.Equal(t, cm.GetResourceVersion(), "")

	err = cli.Get(ctx, cm, nil)
	assert.Error(t, err, "expecting not found error")
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	created := &ConfigMap{}
	err = cli.Create(ctx, cm, created)
	assert.NoError(t, err)
	assert.NotEqual(t, created.GetResourceVersion(), "")

	created.Labels = map[string]string{"foo": "bar"}
	updated := &ConfigMap{}
	err = cli.Update(ctx, created, updated)
	assert.NoError(t, err)

	gotten := &ConfigMap{}
	err = cli.Get(ctx, cm, gotten)
	assert.NoError(t, err)
	assert.Equal(t, gotten.GetName(), cm.GetName())
	assert.Equal(t, gotten.Labels["foo"], "bar")

	err = cli.Delete(ctx, cm, nil)
	assert.NoError(t, err)

	err = cli.Get(ctx, cm, nil)
	assert.Error(t, err, "expecting not found error")
	assert.True(t, IsNotFound(err), "expecting not found error")
}

func TestList(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	namespaces := make([]*Namespace, 0)

	err = cli.List(ctx, Query{Kind: "namespaces"}, &namespaces)
	assert.NoError(t, err)

	// we know there should be at least the default namespace and
	// the kube-system namespace
	assert.True(t, len(namespaces) > 0)

	found := false
	for _, ns := range namespaces {
		if ns.GetName() == "default" {
			found = true
			break
		}
	}

	assert.True(t, found)
}

func TestListSelector(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	myns := &Namespace{
		TypeMeta: TypeMeta{
			Kind: "namespace",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-list-selector-namespace",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	err = cli.Create(ctx, myns, myns)
	assert.NoError(t, err)

	namespaces := make([]*Namespace, 0)

	err = cli.List(ctx, Query{Kind: "namespaces", LabelSelector: "foo=bar"}, &namespaces)
	assert.NoError(t, err)

	assert.Equal(t, len(namespaces), 1)

	if len(namespaces) == 1 {
		assert.Equal(t, namespaces[0].GetName(), myns.GetName())
	}

	err = cli.Delete(ctx, myns, myns)
	assert.NoError(t, err)
}

func TestShortcut(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "cm",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-shortcut-configmap",
		},
	}

	created := &ConfigMap{}
	err = cli.Create(ctx, cm, created)
	assert.NoError(t, err)

	err = cli.Delete(ctx, created, nil)
	assert.NoError(t, err)
}
