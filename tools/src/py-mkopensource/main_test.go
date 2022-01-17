package main

import (
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestGold(t *testing.T) {
	//Arrange
	pipDependencies, err := os.Open("./testdata/pipDependencies.txt")
	require.NoError(t, err)
	defer func() { _ = pipDependencies.Close() }()

	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)

	// Act
	err = Main(markdownOutputType, pipDependencies, w)
	require.NoError(t, err)

	// Assert
	_ = w.Close()
	programOutput, readErr := io.ReadAll(r)
	require.NoError(t, readErr)

	expectedOutput := getFileContents(t, "./testdata/expectedMarkdownOutput.txt")
	require.Equal(t, string(expectedOutput), string(programOutput))

}

func getFileContents(t *testing.T, path string) []byte {
	expErr, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	return expErr
}
