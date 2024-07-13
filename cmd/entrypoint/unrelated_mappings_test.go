package entrypoint_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
)

func getWorkload1MappingGroup(ir *entrypoint.IR) (*entrypoint.IRGroup, bool) {
	for _, group := range ir.Groups {
		if group.Name == "GROUP: workload1-mapping" {
			return &group, true
		}
	}

	return nil, false
}

func predicate(ir *entrypoint.IR) bool {
	_, ok := getWorkload1MappingGroup(ir)

	return ok
}

// The Fake struct is a test harness for Emissary. See testutil_fake_test.go
// for details. Note that this test depends on diagd being in your path. If
// diagd is not available, the test will be skipped.

func TestUnrelatedMappings(t *testing.T) {
	// Use RunFake() to spin up the fake control plane, and note that we
	// _must_ set EnvoyConfig true to do anything with IR. We need the IR
	// for this test, so...
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	// Next up, upsert our test data.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/service.yaml"))
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/host.yaml"))
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/mapping.yaml"))

	// Flush the Fake harness so that we get a configuration.
	f.Flush()

	// We need the IR from that configuration.
	ir, err := f.GetIR(predicate)
	require.NoError(t, err)

	// Now we can check the IR.
	checkIR(ir, t)

	// Next up, upsert a completely unrelated mapping. This mustn't affect
	// our existing group.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/unrelated.yaml"))

	// Flush the Fake harness and repeat our IR check.
	f.Flush()
	ir, err = f.GetIR(predicate)
	require.NoError(t, err)
	checkIR(ir, t)
}

func checkIR(ir *entrypoint.IR, t *testing.T) {
	// In the IR, we should find a group called "workload1-mapping".
	group, ok := getWorkload1MappingGroup(ir)
	require.True(t, ok)

	// That group should have two mappings.
	require.Len(t, group.Mappings, 2)

	// One mapping should have a "name" of "workload1-mapping" and a "_weight"
	// of 100; the other should have a "name" of "workload2-mapping" and a
	// "_weight" of 10.
	found1 := false
	found2 := false

	for _, mapping := range group.Mappings {
		switch mapping.Name {
		case "workload1-mapping":
			assert.Equal(t, 100, mapping.Weight)
			found1 = true
		case "workload2-mapping":
			assert.Equal(t, 10, mapping.Weight)
			found2 = true
		default:
			t.Fatalf("unexpected mapping: %#v", mapping)
		}
	}

	assert.True(t, found1)
	assert.True(t, found2)
}
