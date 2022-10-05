package entrypoint

import (
	"context"
	"os"
	"testing"

	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/stretchr/testify/assert"
)

type queryTest struct {
	name          string
	q             map[string]thingToWatch
	fieldSelector string
	labelSelector string
	expect        []kates.Query
}

func TestInterestingTypesQueryies(t *testing.T) {
	tests := []queryTest{
		{
			name: "build simple query",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
			},
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "", LabelSelector: ""},
			},
		},
		{
			name: "Only select secrets of type tls",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
				"Secrets": {
					typename: "secrets.v1",
				},
			},
			fieldSelector: "secrets.v1:type=kubernetes.io/tls",
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "", LabelSelector: ""},
				{Name: "Secrets", Kind: "secrets.v1", Namespace: "", FieldSelector: "type=kubernetes.io/tls", LabelSelector: ""},
			},
		},
		{
			name: "Only select secrets of type tls and version: v2 label",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
				"Secrets": {
					typename: "secrets.v1",
				},
			},
			fieldSelector: "secrets.v1:type=kubernetes.io/tls",
			labelSelector: "secrets.v1:version=v2",
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "", LabelSelector: ""},
				{Name: "Secrets", Kind: "secrets.v1", Namespace: "", FieldSelector: "type=kubernetes.io/tls", LabelSelector: "version=v2"},
			},
		},
		{
			name: "Only select resources labeled by version: v2 and specific fields",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
				"Secrets": {
					typename: "secrets.v1",
				},
			},
			fieldSelector: "secrets.v1:type=kubernetes.io/tls;resourcegroup.v1:metadata.name=name",
			labelSelector: "secrets.v1:version=v2;resourcegroup.v1:version=v2",
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "metadata.name=name", LabelSelector: "version=v2"},
				{Name: "Secrets", Kind: "secrets.v1", Namespace: "", FieldSelector: "type=kubernetes.io/tls", LabelSelector: "version=v2"},
			},
		},
		{
			name: "Combine resourcegroup and generic selector",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
				"Secrets": {
					typename: "secrets.v1",
				},
			},
			labelSelector: "secrets.v1:version=v2;version=v1",
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "", LabelSelector: "version=v1"},
				{Name: "Secrets", Kind: "secrets.v1", Namespace: "", FieldSelector: "", LabelSelector: "version=v2"},
			},
		},
		{
			name: "resourcegroup selector is weighted more than a generic selector",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
				"Secrets": {
					typename: "secrets.v1",
				},
			},
			labelSelector: "version=v1;secrets.v1:version=v2",
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "", LabelSelector: "version=v1"},
				{Name: "Secrets", Kind: "secrets.v1", Namespace: "", FieldSelector: "", LabelSelector: "version=v2"},
			},
		},
		{
			name: "Selector without specific apiversion",
			q: map[string]thingToWatch{
				"Resource": {
					typename: "resourcegroup.v1",
				},
				"Secrets": {
					typename: "secrets.v1",
				},
			},
			labelSelector: "secrets:version=v1",
			expect: []kates.Query{
				{Name: "Resource", Kind: "resourcegroup.v1", Namespace: "", FieldSelector: "", LabelSelector: ""},
				{Name: "Secrets", Kind: "secrets.v1", Namespace: "", FieldSelector: "", LabelSelector: "version=v1"},
			},
		},
	}

	defer func() {
		os.Setenv("AMBASSADOR_WATCHER_FIELD_SELECTOR", "")
		os.Setenv("AMBASSADOR_WATCHER_LABEL_SELECTOR", "")
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("AMBASSADOR_WATCHER_FIELD_SELECTOR", test.fieldSelector)
			os.Setenv("AMBASSADOR_WATCHER_LABEL_SELECTOR", test.labelSelector)
			queries := GetQueries(context.TODO(), test.q)

			for _, gotQuery := range queries {
				for _, expectQuery := range test.expect {
					if expectQuery.Name != gotQuery.Name {
						continue
					}

					assert.Equal(t, expectQuery.Kind, gotQuery.Kind)
					assert.Equal(t, expectQuery.Namespace, gotQuery.Namespace)
					assert.Equal(t, expectQuery.FieldSelector, gotQuery.FieldSelector)
					assert.Equal(t, expectQuery.LabelSelector, gotQuery.LabelSelector)
				}
			}

			assert.Equal(t, len(test.expect), len(queries))
		})
	}
}
