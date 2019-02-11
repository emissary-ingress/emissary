package pgs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoFmt_Match(t *testing.T) {
	t.Parallel()

	pp := GoFmt()

	tests := []struct {
		n string
		a Artifact
		m bool
	}{
		{"GenFile", GeneratorFile{Name: "foo.go"}, true},
		{"GenFileNonGo", GeneratorFile{Name: "bar.txt"}, false},

		{"GenTplFile", GeneratorTemplateFile{Name: "foo.go"}, true},
		{"GenTplFileNonGo", GeneratorTemplateFile{Name: "bar.txt"}, false},

		{"CustomFile", CustomFile{Name: "foo.go"}, true},
		{"CustomFileNonGo", CustomFile{Name: "bar.txt"}, false},

		{"CustomTplFile", CustomTemplateFile{Name: "foo.go"}, true},
		{"CustomTplFileNonGo", CustomTemplateFile{Name: "bar.txt"}, false},

		{"NonMatch", GeneratorAppend{FileName: "foo.go"}, false},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.n, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.m, pp.Match(tc.a))
		})
	}
}

func TestGoFmt_Process(t *testing.T) {
	t.Parallel()

	src := []byte("// test\n package           foo\n\nvar          bar          int = 123\n")
	exp := []byte("// test\npackage foo\n\nvar bar int = 123\n")

	out, err := GoFmt().Process(src)
	assert.NoError(t, err)
	assert.Equal(t, exp, out)
}

type mockPP struct {
	match bool
	out   []byte
	err   error
}

func (pp mockPP) Match(a Artifact) bool             { return pp.match }
func (pp mockPP) Process(in []byte) ([]byte, error) { return pp.out, pp.err }
