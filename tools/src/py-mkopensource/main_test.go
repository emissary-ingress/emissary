package main

import (
	"encoding/json"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path"
	"testing"
)

func TestMarkdownOutput(t *testing.T) {
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
	require.Equal(t, expectedOutput, string(programOutput))
}

func TestJsonOutput(t *testing.T) {
	//Arrange
	pipDependencies, err := os.Open("./testdata/successful-generation/dependency_list.txt")
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
	expectedOutput := getDependencyInfoFromFile(t, "./testdata/successful-generation/expected_json.json")
	require.Equal(t, expectedOutput, programOutput)
}

func TestLicenseErrors(t *testing.T) {
	testCases := []struct {
		testName     string
		dependencies string
		outputType   OutputType
	}{
		{
			"GPL licenses are forbidden - Markdown format",
			"./testdata/gpl-license",
			markdownOutputType,
		},
		{
			"GPL licenses are forbidden - JSON format",
			"./testdata/gpl-license",
			jsonOutputType,
		},
		{
			"AGPL licenses are forbidden - Markdown format",
			"./testdata/agpl-license",
			markdownOutputType,
		},
		{
			"AGPL licenses are forbidden - JSON format",
			"./testdata/agpl-license",
			jsonOutputType,
		},
		{
			"Unknown licenses are identified correctly",
			"./testdata/unknown-license",
			jsonOutputType,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			//Arrange
			pipDependencies, err := os.Open(path.Join(testCase.dependencies, "dependency_list.txt"))
			require.NoError(t, err)
			defer func() { _ = pipDependencies.Close() }()

			_, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)

			// Act
			err = Main(markdownOutputType, pipDependencies, w)
			require.Error(t, err)
			expectedError := getFileContents(t, path.Join(testCase.dependencies, "expected_err.txt"))
			require.Equal(t, expectedError, err.Error())
			_ = w.Close()
		})
	}
}

func getFileContents(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil && err != io.EOF {
		require.NoError(t, err)
	}
	return string(content)
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
