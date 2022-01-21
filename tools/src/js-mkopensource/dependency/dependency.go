package dependency

import (
	"encoding/json"
	"fmt"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	"github.com/datawire/go-mkopensource/pkg/detectlicense"
	"io"
	"regexp"
	"sort"
	"strings"
)

type NodeDependencies map[string]nodeDependency

type nodeDependency struct {
	Licenses       string `json:"licenses"`
	Repository     string `json:"repository"`
	DependencyPath string `json:"dependencyPath"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Path           string `json:"path"`
	URL            string `json:"url"`
	LicenseFile    string `json:"licenseFile"`
	LicenseText    string `json:"licenseText"`
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

	dependency := &dependencies.Dependency{
		Name:     name,
		Version:  version,
		Licenses: []string{},
	}

	allLicenses, err := getDependencyLicenses(dependencyId, nodeDependency)
	if err != nil {
		return nil, err
	}
	dependency.Licenses = allLicenses

	return dependency, nil
}

func getDependencyLicenses(dependencyId string, nodeDependency nodeDependency) ([]string, error) {
	parenthesisRe, err := regexp.Compile(`^\(|\)$`)
	if err != nil {
		return nil, err
	}
	licenseString := parenthesisRe.ReplaceAllString(nodeDependency.Licenses, "")

	separatorRe, err := regexp.Compile(` OR | AND `)
	if err != nil {
		return nil, err
	}
	licenses := separatorRe.Split(licenseString, -1)

	allLicenses := []string{}
	for _, spdxId := range licenses {
		license, ok := detectlicense.SpdxIdentifiers[spdxId]
		if ok {
			allLicenses = append(allLicenses, license.Name)
			continue
		}

		licenses, ok := hardcodedDependencies[dependencyId]
		if ok {
			allLicenses = licenses
			break
		}

		return nil, fmt.Errorf("\nFound an unknown SPDX Identifier '%s'.\n"+
			"Dependecy name: %s@%s\n"+
			"Dependecy URL: %s\n"+
			"License text:\n%#v\n", nodeDependency.Licenses, nodeDependency.Name, nodeDependency.Version,
			nodeDependency.URL, nodeDependency.LicenseText)
	}

	sort.Strings(allLicenses)
	return allLicenses, nil
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
