package entrypoint_test

import (
	"fmt"
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

type WeightCheck struct {
	weight     *int
	cumulative int
}

func NewWeightCheck(weight int, cumulative int) WeightCheck {
	weightPtr := &weight

	if weight < 0 {
		weightPtr = nil
	}

	return WeightCheck{weight: weightPtr, cumulative: cumulative}
}

// checkIR is a helper function that flushes the world, gets an IR, then
// checks the IR for the expected state. It'll find the "workload1-mapping"
// group and check that it has mappings for each entry in the weights map,
// with the correct weights.
func checkIR(f *entrypoint.Fake, what string, weights map[string]WeightCheck) {
	// Flush the Fake harness so that we get a configuration.
	f.Flush()

	// We need the IR from that configuration.
	ir, err := f.GetIR(predicate)
	require.NoError(f.T, err)

	// In the IR, we should find a group called "workload1-mapping".
	group, ok := getWorkload1MappingGroup(ir)
	require.True(f.T, ok)

	// That group should have the same number of mappings as we have entries
	// in the weights map.
	require.Len(f.T, group.Mappings, len(weights))

	// Now we can check each mapping. Since we need all of them to be present
	// in the group, we'll start with a set of all the mappings defined in the
	// weights map, and remove them as we find them in the mapping. Any left
	// over at the end were missing from the group.
	missingMappings := make(map[string]struct{})
	for name := range weights {
		missingMappings[name] = struct{}{}
	}

	// Next, walk over the group's mappings and check against the weights map.
	for _, mapping := range group.Mappings {
		check, ok := weights[mapping.Name]

		if ok {
			// It's present; remove it from the leftovers.
			delete(missingMappings, mapping.Name)

			// Next, make sure the weights match.
			var msg string

			if check.weight == nil {
				if mapping.Weight != nil {
					msg = fmt.Sprintf("%s: weight for %s should not be present but is %d", what, mapping.Name, *mapping.Weight)
				}
			} else if mapping.Weight == nil {
				msg = fmt.Sprintf("%s: weight for %s should be %d but is not present", what, mapping.Name, *check.weight)
			} else if *check.weight != *mapping.Weight {
				msg = fmt.Sprintf("%s: unexpected weight for mapping %s: wanted %d, got %d", what, mapping.Name, *check.weight, mapping.Weight)
			}

			if msg != "" {
				for _, m := range group.Mappings {
					msg += "\n"

					if m.Weight == nil {
						msg += fmt.Sprintf("  - %s: weight unset", m.Name)
					} else {
						msg += fmt.Sprintf("  - %s: weight %d", m.Name, *m.Weight)
					}
				}

				f.T.Fatal(msg)
			}

			// Finally, check the cumulative weight.
			if check.cumulative != mapping.CumulativeWeight {
				f.T.Fatalf("%s: unexpected cumulative weight for mapping %s: wanted %d, got %d", what, mapping.Name, check.cumulative, mapping.CumulativeWeight)
			}
		} else {
			// It's not present; this is a problem.
			f.T.Fatalf("%s: unexpected mapping: %#v", what, mapping.Name)
		}
	}

	// Finally, we should have no leftovers.
	if len(missingMappings) > 0 {
		msg := fmt.Sprintf("%s: missing mappings:", what)

		for name := range missingMappings {
			msg += "\n"
			msg += fmt.Sprintf("  - %s", name)
		}

		f.T.Fatal(msg)
	}
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
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/mapping1.yaml"))

	// Now we can check the IR.
	checkIR(f, "initial", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(-1, 100),
		"workload2-mapping": NewWeightCheck(10, 10),
	})

	// Next up, upsert a completely unrelated mapping. This mustn't affect
	// our existing group.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/unrelated.yaml"))

	checkIR(f, "upsert unrelated", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(-1, 100),
		"workload2-mapping": NewWeightCheck(10, 10),
	})

	// Next, try updating the weight of workload2-mapping.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/mapping2.yaml"))

	checkIR(f, "update workload2-mapping weight", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(-1, 100),
		"workload2-mapping": NewWeightCheck(50, 50),
	})

	// Next up, delete our completely unrelated mapping. This mustn't affect
	// our existing group.
	assert.NoError(t, f.Delete("Mapping", "infrastructure", "unrelated"))

	checkIR(f, "delete unrelated", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(-1, 100),
		"workload2-mapping": NewWeightCheck(50, 50),
	})

	// Repeat that upsert-and-repeat cycle.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/unrelated.yaml"))

	checkIR(f, "re-upsert unrelated", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(-1, 100),
		"workload2-mapping": NewWeightCheck(50, 50),
	})

	assert.NoError(t, f.Delete("Mapping", "infrastructure", "unrelated"))

	checkIR(f, "re-delete unrelated", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(-1, 100),
		"workload2-mapping": NewWeightCheck(50, 50),
	})

	// Finally, do something complex: update the weight of workload1-mapping,
	// add a workload3-mapping, and reintroduce the unrelated mapping.
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/mapping3.yaml"))
	assert.NoError(t, f.UpsertFile("testdata/unrelated-mappings/unrelated.yaml"))

	checkIR(f, "complex 1", map[string]WeightCheck{
		"workload1-mapping": NewWeightCheck(20, 20),
		"workload2-mapping": NewWeightCheck(50, 70),
		"workload3-mapping": NewWeightCheck(-1, 100),
	})
}
