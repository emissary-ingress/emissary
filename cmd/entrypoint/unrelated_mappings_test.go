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

func checkIR(f *entrypoint.Fake) {
	// Flush the Fake harness so that we get a configuration.
	f.Flush()

	// We need the IR from that configuration.
	ir, err := f.GetIR(predicate)
	require.NoError(f.T, err)

	// In the IR, we should find a group called "workload1-mapping".
	group, ok := getWorkload1MappingGroup(ir)
	require.True(f.T, ok)

	// That group should have two mappings.
	require.Len(f.T, group.Mappings, 2)

	// One mapping should have a "name" of "workload1-mapping" and a
	// cumulative weight of 100; the other should have a "name" of
	// "workload2-mapping" and a cumulative weight of 10.
	found1 := false
	found2 := false

	for _, mapping := range group.Mappings {
		switch mapping.Name {
		case "workload1-mapping":
			assert.Equal(f.T, 100, mapping.CumulativeWeight)
			found1 = true
		case "workload2-mapping":
			assert.Equal(f.T, 10, mapping.CumulativeWeight)
			found2 = true
		default:
			f.T.Fatalf("unexpected mapping: %#v", mapping)
		}
	}

	assert.True(f.T, found1)
	assert.True(f.T, found2)
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

	// Now we can check the IR.
	checkIR(f)

	// Next up, upsert a completely unrelated mapping. This mustn't affect
	// our existing group.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/unrelated.yaml"))

	// Flush the Fake harness and repeat our IR check.
	checkIR(f)
}
