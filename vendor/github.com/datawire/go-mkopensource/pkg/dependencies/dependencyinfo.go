package dependencies

import (
	"encoding/json"
	"fmt"
	"github.com/datawire/go-mkopensource/pkg/detectlicense"
)

//nolint:gochecknoglobals // Can't be a constant
var knownLicenses = map[string]detectlicense.License{
	detectlicense.AmbassadorProprietary.Name: detectlicense.AmbassadorProprietary,
	detectlicense.PublicDomain.Name:          detectlicense.PublicDomain,
	detectlicense.Apache2.Name:               detectlicense.Apache2,
	detectlicense.BSD1.Name:                  detectlicense.BSD1,
	detectlicense.BSD2.Name:                  detectlicense.BSD2,
	detectlicense.BSD3.Name:                  detectlicense.BSD3,
	detectlicense.CcBySa40.Name:              detectlicense.CcBySa40,
	detectlicense.GPL3.Name:                  detectlicense.GPL3,
	detectlicense.ISC.Name:                   detectlicense.ISC,
	detectlicense.LGPL21OrLater.Name:         detectlicense.LGPL21OrLater,
	detectlicense.MIT.Name:                   detectlicense.MIT,
	detectlicense.MPL2.Name:                  detectlicense.MPL2,
	detectlicense.PSF.Name:                   detectlicense.PSF,
	detectlicense.Unicode2015.Name:           detectlicense.Unicode2015}

type DependencyInfo struct {
	Dependencies []Dependency      `json:"dependencies"`
	Licenses     map[string]string `json:"licenseInfo"`
}

type Dependency struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Licenses []string `json:"licenses"`
}

func NewDependencyInfo() DependencyInfo {
	return DependencyInfo{
		Dependencies: []Dependency{},
		Licenses:     map[string]string{},
	}
}

func (d *DependencyInfo) Unmarshal(data []byte) error {
	if err := json.Unmarshal(data, d); err != nil {
		return err
	}

	return nil
}

func (d *DependencyInfo) UpdateLicenseList() error {
	usedLicenses := map[string]detectlicense.License{}

	for _, dependency := range d.Dependencies {
		for _, licenseName := range dependency.Licenses {
			license, ok := knownLicenses[licenseName]
			if !ok {
				return fmt.Errorf("license details for '%s' are not known", licenseName)
			}
			usedLicenses[license.Name] = license
		}
	}

	for k, v := range usedLicenses {
		d.Licenses[k] = v.URL
	}

	return nil
}
