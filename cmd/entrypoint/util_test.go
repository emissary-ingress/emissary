package entrypoint

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// loadTestCases is a generic function for loading test cases from yaml
// files in a directory.
//
// Define a type of the generic T as the structure of the files you are loading. Each file
// then becomes a test case where you can place all your inputs and
// expectations.
func loadTestCases[T any](t *testing.T, dirPath, pattern string) []*T {
	t.Helper()

	tcFiles, err := fs.Glob(os.DirFS(dirPath), pattern)
	require.NoError(t, err)
	require.Greater(t, len(tcFiles), 0)

	var testCases []*T

	for _, f := range tcFiles {
		filePath := filepath.Join(dirPath, f)
		fileData, err := os.ReadFile(filePath)
		require.NoError(t, err)

		tc := new(T)

		err = yaml.UnmarshalStrict(fileData, tc)
		require.NoError(t, err)

		testCases = append(testCases, tc)
	}

	return testCases
}
