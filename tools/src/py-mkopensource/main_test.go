package main

import (
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestGold(t *testing.T) {
	testCases := []struct {
		testName       string
		outputTypeFlag OutputType
		expectedOutput string
	}{
		{
			"Markdown output",
			markdownOutputType,
			"./testdata/successful-generation/expected_markdown.txt",
		},
		{
			"Json output",
			jsonOutputType,
			"./testdata/successful-generation/expected_json.json",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			//Arrange
			pipDependencies, err := os.Open("./testdata/successful-generation/dependency_list.txt")
			require.NoError(t, err)
			defer func() { _ = pipDependencies.Close() }()

			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)

			// Act
			err = Main(testCase.outputTypeFlag, pipDependencies, w)
			require.NoError(t, err)

			// Assert
			_ = w.Close()
			programOutput, readErr := io.ReadAll(r)
			require.NoError(t, readErr)

			expectedOutput := getFileContents(t, testCase.expectedOutput)
			require.Equal(t, string(expectedOutput), string(programOutput))
		})
	}
}

func getFileContents(t *testing.T, path string) []byte {
	expErr, err := os.ReadFile(path)
	if err != nil && err != io.EOF {
		require.NoError(t, err)
	}
	return expErr
}
