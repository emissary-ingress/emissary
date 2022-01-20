package dependency

import (
	"encoding/json"
	"fmt"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	"github.com/datawire/go-mkopensource/pkg/detectlicense"
	"io"
	"sort"
	"strings"
)

type NodeDependencies map[string]nodeDependency

type nodeDependency struct {
	Licenses string `json:"licenses"`
}

func GetDependencyInformation(r io.Reader) (dependencyInfo dependencies.DependencyInfo, err error) {
	nodeDependencies := &NodeDependencies{}
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, nodeDependencies)
	if err != nil {
		return
	}

	sortedDependencies := getSortedDependencies(nodeDependencies)

	dependencyInfo = dependencies.NewDependencyInfo()
	for _, dependencyId := range sortedDependencies {
		nodeDependency := (*nodeDependencies)[dependencyId]

		dependency, err := getDependencyDetails(nodeDependency, dependencyId)
		if err != nil {
			return dependencyInfo, err
		}

		dependencyInfo.Dependencies = append(dependencyInfo.Dependencies, *dependency)
	}

	err = dependencyInfo.UpdateLicenseList()
	if err != nil {
		return
	}

	return
}

func getDependencyDetails(nodeDependency nodeDependency, dependencyId string) (*dependencies.Dependency, error) {
	name, version := splitDependencyIdentifier(dependencyId)
	license, ok := detectlicense.SpdxIdentifiers[nodeDependency.Licenses]
	if !ok {
		return nil, fmt.Errorf("there is no license information for SPDX Identifier '%s' used by %s", nodeDependency.Licenses, dependencyId)
	}

	dependency := &dependencies.Dependency{
		Name:     name,
		Version:  version,
		Licenses: []string{license.Name},
	}

	return dependency, nil
}

func getSortedDependencies(nodeDependencies *NodeDependencies) []string {
	sortedDependencies := make([]string, 0, len(*nodeDependencies))
	for k := range *nodeDependencies {
		sortedDependencies = append(sortedDependencies, k)
	}
	sort.Strings(sortedDependencies)
	return sortedDependencies
}

func splitDependencyIdentifier(identifier string) (name string, version string) {
	parts := strings.Split(identifier, "@")

	numberOfParts := len(parts)
	return strings.Join(parts[:numberOfParts-1], "@"), parts[numberOfParts-1]
}
