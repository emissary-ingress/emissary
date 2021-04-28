package entrypoint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type obj struct{}

func TestParseAnnotations(t *testing.T) {
	ctx := context.Background()
	mapping := `
---
apiVersion: getambassador.io/v2
kind: Mapping
name: quote-backend
prefix: /backend/
precedence: 10
service: quote:80
timeout_ms: 10000
resolver: endpoint
load_balancer:
  policy: round_robin
`

	svc := &kates.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": mapping,
			},
		},
	}

	res := GetAnnotations(ctx, svc)

	for _, r := range res {
		switch v := r.(type) {
		case *amb.Mapping:
			mapping := r.(*amb.Mapping)

			assert.Equal(t, mapping.Spec.Prefix, "/backend/")
			assert.Equal(t, mapping.Spec.Resolver, "endpoint")
		default:
			t.Fatalf("got unexpected type %+v", v)
		}
	}
}
