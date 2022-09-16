package main

import (
	"fmt"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	"github.com/datawire/go-mkopensource/pkg/detectlicense"
	"github.com/datawire/go-mkopensource/pkg/golist"
	"sort"
)

func GenerateDependencyList(modNames []string, modLicenses map[string]map[detectlicense.License]struct{},
	modInfos map[string]*golist.Module, goVersion string,
	licenseRestriction detectlicense.LicenseRestriction) (dependencyList dependencies.DependencyInfo, errors []error) {
	dependencyList = dependencies.NewDependencyInfo()
	errors = []error{}

	for _, modKey := range modNames {
		ambassadorProprietary := isAmbassadorProprietary(modLicenses[modKey])
		if ambassadorProprietary {
			continue
		}

		modVal := modInfos[modKey]

		dependencyDetails := dependencies.Dependency{
			Name:     getDependencyName(modVal),
			Version:  getDependencyVersion(modVal, goVersion),
			Licenses: []string{},
		}

		for license := range modLicenses[modKey] {
			dependencyDetails.Licenses = append(dependencyDetails.Licenses, license.Name)

			if err := dependencies.CheckLicenseRestrictions(dependencyDetails, license.Name, licenseRestriction); err != nil {
				errors = append(errors, err)
			}
		}
		sort.Strings(dependencyDetails.Licenses)

		dependencyList.Dependencies = append(dependencyList.Dependencies, dependencyDetails)
	}

	if err := dependencyList.UpdateLicenseList(); err != nil {
		errors = append(errors, fmt.Errorf("Could not generate list of license URLs: %v\n", err))
	}

	return dependencyList, errors
}

func getDependencyName(modVal *golist.Module) string {
	if modVal == nil {
		return "the Go language standard library (\"std\")"
	}

	if modVal.Replace != nil && modVal.Replace.Version != "" && modVal.Replace.Path != modVal.Path {
		return fmt.Sprintf("%s (modified from %s)", modVal.Replace.Path, modVal.Path)
	}

	return modVal.Path
}

func getDependencyVersion(modVal *golist.Module, goVersion string) string {
	if modVal == nil {
		return goVersion
	}

	if modVal.Replace != nil {
		if modVal.Replace.Version == "" {
			return "(modified)"
		} else {
			return modVal.Replace.Version
		}
	}

	return modVal.Version
}
