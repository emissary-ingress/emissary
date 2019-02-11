package pgs

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in []interface{}
		ex string
	}{
		{
			[]interface{}{"foo", " bar", " baz"},
			"// foo bar baz\n",
		},
		{
			in: []interface{}{"the quick brown fox jumps over the lazy dog"},
			ex: "// the quick brown\n// fox jumps over\n// the lazy dog\n",
		},
		{
			in: []interface{}{"supercalifragilisticexpialidocious"},
			ex: "// supercalifragilisticexpialidocious\n",
		},
		{
			in: []interface{}{"1234567890123456789012345 foo"},
			ex: "// 1234567890123456789012345\n// foo\n",
		},
	}

	for i, test := range tests {
		tc := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tc.ex, C(20, tc.in...))
		})
	}
}

func TestC80(t *testing.T) {
	t.Parallel()
	ex := "// foo foo foo foo foo foo foo foo foo foo foo foo foo foo foo foo foo foo foo\n// foo\n"
	assert.Equal(t, ex, C80(strings.Repeat("foo ", 20)))
}
