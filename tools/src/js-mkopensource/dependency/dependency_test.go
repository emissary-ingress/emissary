package dependency_test

import (
	"github.com/datawire/ambassador/v2/tools/src/js-mkopensource/dependency"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestSuccessfulGeneration(t *testing.T) {
	testCases := []struct {
		testName       string
		input          string
		expectedOutput string
	}{
		{
			"Dependency identifier in the format @name@version",
			"./testdata/dependency-with-special-characters/dependencies.json",
			"./testdata/dependency-with-special-characters/expected_output.json",
		},
		{
			"Multiple dependencies",
			"./testdata/multiple-licenses/dependencies.json",
			"./testdata/multiple-licenses/expected_output.json",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			//Arrange
			nodeDependencies := getNodeDependencies(t, testCase.input)
			defer func() { _ = nodeDependencies.Close() }()

			// Act
			dependencyInformation, err := dependency.GetDependencyInformation(nodeDependencies)
			require.NoError(t, err)

			// Assert
			expectedJson := getDependencyInfoFromFile(t, testCase.expectedOutput)
			require.Equal(t, *expectedJson, dependencyInformation)
		})
	}
}

func TestErrorScenarios(t *testing.T) {
	testCases := []struct {
		testName string
		input    string
	}{
		{
			"Invalid Json input",
			"./testdata/invalid-json/dependencies.json",
		},
		{
			"Unknown license identifier",
			"./testdata/unknown-license/dependencies.json",
		},
		{
			"Missing license",
			"./testdata/missing-license/dependencies.json",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			//Arrange
			nodeDependencies := getNodeDependencies(t, testCase.input)
			defer func() { _ = nodeDependencies.Close() }()

			// Act
			_, err := dependency.GetDependencyInformation(nodeDependencies)

			// Assert
			require.Error(t, err)
		})
	}
}

func getNodeDependencies(t *testing.T, dependencyFile string) *os.File {
	nodeDependencies, openErr := os.Open(dependencyFile)
	require.NoError(t, openErr)
	return nodeDependencies
}

func getDependencyInfoFromFile(t *testing.T, path string) *dependencies.DependencyInfo {
	f, openErr := os.Open(path)
	require.NoError(t, openErr)

	data, readErr := io.ReadAll(f)
	require.NoError(t, readErr)

	dependencyInfo := &dependencies.DependencyInfo{}
	unmarshalErr := dependencyInfo.Unmarshal(data)
	require.NoError(t, unmarshalErr)

	return dependencyInfo
}
