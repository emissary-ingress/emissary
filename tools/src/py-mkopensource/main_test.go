package main

import (
	"encoding/json"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestMarkdownOutput(t *testing.T) {
	testCases := []struct {
		testName       string
		dependencies   string
		expectedOutput string
	}{
		{
			"Different dependencies are processed correctly",
			"./testdata/successful-generation/dependency_list.txt",
			"./testdata/successful-generation/expected_markdown.json",
		},
		{
			"Same dependency twice with different version",
			"./testdata/two-versions-of-a-dependency/dependency_list.txt",
			"./testdata/two-versions-of-a-dependency/expected_markdown.json",
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
			err = Main(markdownOutputType, pipDependencies, w)
			require.NoError(t, err)
			_ = w.Close()

			// Assert
			programOutput, readErr := io.ReadAll(r)
			require.NoError(t, readErr)

			expectedOutput := getFileContents(t, "./testdata/successful-generation/expected_markdown.txt")
			require.Equal(t, string(expectedOutput), string(programOutput))
		})
	}
}

func TestJsonOutput(t *testing.T) {
	testCases := []struct {
		testName       string
		dependencies   string
		expectedOutput string
	}{
		{
			"Different dependencies are processed correctly",
			"./testdata/successful-generation/dependency_list.txt",
			"./testdata/successful-generation/expected_json.json",
		},
		{
			"Same dependency twice with different version",
			"./testdata/two-versions-of-a-dependency/dependency_list.txt",
			"./testdata/two-versions-of-a-dependency/expected_json.json",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			//Arrange
			pipDependencies, err := os.Open(testCase.dependencies)
			require.NoError(t, err)
			defer func() { _ = pipDependencies.Close() }()

			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)

			// Act
			err = Main(jsonOutputType, pipDependencies, w)
			require.NoError(t, err)
			_ = w.Close()

			// Assert
			programOutput := getDependencyInfoFromReader(t, r)
			expectedOutput := getDependencyInfoFromFile(t, testCase.expectedOutput)
			require.Equal(t, expectedOutput, programOutput)
		})
	}
}

func getFileContents(t *testing.T, path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil && err != io.EOF {
		require.NoError(t, err)
	}
	return content
}

func getDependencyInfoFromFile(t *testing.T, path string) *dependencies.DependencyInfo {
	f, err := os.Open(path)
	require.NoError(t, err)

	return getDependencyInfoFromReader(t, f)
}

func getDependencyInfoFromReader(t *testing.T, r io.Reader) *dependencies.DependencyInfo {
	data, readErr := io.ReadAll(r)
	require.NoError(t, readErr)

	jsonOutput := &dependencies.DependencyInfo{}
	err := json.Unmarshal(data, jsonOutput)
	require.NoError(t, err)

	return jsonOutput
}
