// MODIFIED: This file is copied verbatim from Go 1.15.5 os/exec/bench_test.go,
// MODIFIED: except for lines marked "MODIFIED".
//
// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dexec // MODIFIED

import (
	"testing"

	"github.com/datawire/ambassador/pkg/dlog" // MODIFIED
)

func BenchmarkExecHostname(b *testing.B) {
	ctx := dlog.NewTestContext(b, false) // MODIFIED
	b.ReportAllocs()
	path, err := LookPath("hostname")
	if err != nil {
		b.Fatalf("could not find hostname: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := CommandContext(ctx, path).Run(); err != nil { // MODIFIED
			b.Fatalf("hostname: %v", err)
		}
	}
}
