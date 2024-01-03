package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

type NamespaceCtxKey string

// CreateNSForTest creates a random namespace with the runID as a prefix. It is stored in the context
// so that the deleteNSForTest routine can look it up and delete it.
func CreateNSForTest(ctx context.Context, cfg *envconf.Config, t *testing.T, runID string) (context.Context, error) {
	ns := envconf.RandomName(runID, 16)
	ctx = context.WithValue(ctx, GetNamespaceKey(t), ns)

	t.Logf("Creating NS %v for test %v", ns, t.Name())
	nsObj := v1.Namespace{}
	nsObj.Name = ns
	return ctx, cfg.Client().Resources().Create(ctx, &nsObj)
}

// DeleteNSForTest looks up the namespace corresponding to the given test and deletes it.
func DeleteNSForTest(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
	ns := fmt.Sprint(ctx.Value(GetNamespaceKey(t)))
	t.Logf("Deleting NS %v for test %v", ns, t.Name())

	nsObj := v1.Namespace{}
	nsObj.Name = ns
	return ctx, cfg.Client().Resources().Delete(ctx, &nsObj)
}

// GetNamespaceKey returns the context key for a given test
func GetNamespaceKey(t *testing.T) NamespaceCtxKey {
	if strings.Contains(t.Name(), "/") {
		return NamespaceCtxKey(strings.Split(t.Name(), "/")[0])
	}

	return NamespaceCtxKey(t.Name())
}
