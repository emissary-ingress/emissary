package dependencies

import (
	"encoding/json"
	"fmt"
	. "github.com/datawire/go-mkopensource/pkg/detectlicense"
)

//nolint:gochecknoglobals // Can't be a constant
var licensesByName = map[string]License{
	AmbassadorProprietary.Name: AmbassadorProprietary,
	ZeroBSD.Name:               ZeroBSD,
	Apache2.Name:               Apache2,
	AFL21.Name:                 AFL21,
	AGPL1Only.Name:             AGPL1Only,
	AGPL1OrLater.Name:          AGPL1OrLater,
	AGPL3Only.Name:             AGPL3Only,
	AGPL3OrLater.Name:          AGPL3OrLater,
	BSD1.Name:                  BSD1,
	BSD2.Name:                  BSD2,
	BSD3.Name:                  BSD3,
	Cc010.Name:                 Cc010,
	CcBy30.Name:                CcBy30,
	CcBy40.Name:                CcBy40,
	CcBySa40.Name:              CcBySa40,
	EPL10.Name:                 EPL10,
	GPL1Only.Name:              GPL1Only,
	GPL1OrLater.Name:           GPL1OrLater,
	GPL2Only.Name:              GPL2Only,
	GPL2OrLater.Name:           GPL2OrLater,
	GPL3Only.Name:              GPL3Only,
	GPL3OrLater.Name:           GPL3OrLater,
	ISC.Name:                   ISC,
	LGPL2Only.Name:             LGPL2Only,
	LGPL2OrLater.Name:          LGPL2OrLater,
	LGPL21Only.Name:            LGPL21Only,
	LGPL21OrLater.Name:         LGPL21OrLater,
	LGPL3Only.Name:             LGPL3Only,
	LGPL3OrLater.Name:          LGPL3OrLater,
	MIT.Name:                   MIT,
	MPL11.Name:                 MPL11,
	MPL2.Name:                  MPL2,
	ODCBy10.Name:               ODCBy10,
	OFL11.Name:                 OFL11,
	PSF.Name:                   PSF,
	Python20.Name:              Python20,
	PublicDomain.Name:          PublicDomain,
	Unicode2015.Name:           Unicode2015,
	Unlicense.Name:             Unlicense,
	WTFPL.Name:                 WTFPL,
}

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
	usedLicenses := map[string]License{}

	for _, dependency := range d.Dependencies {
		for _, licenseName := range dependency.Licenses {
			license, err := getLicenseFromName(licenseName)
			if err != nil {
				return err
			}
			usedLicenses[license.Name] = license
		}
	}

	for k, v := range usedLicenses {
		d.Licenses[k] = v.URL
	}

	return nil
}

func getLicenseFromName(licenseName string) (License, error) {
	license, ok := licensesByName[licenseName]
	if !ok {
		return License{}, fmt.Errorf("license details for '%s' are not known", licenseName)
	}
	return license, nil
}

func CheckLicenseRestrictions(dependency Dependency, licenseName string, licenseRestriction LicenseRestriction) error {
	license, err := getLicenseFromName(licenseName)
	if err != nil {
		return err
	}

	if license.Restriction == Forbidden {
		return fmt.Errorf("Dependency '%s@%s' uses license '%s' which is forbidden.", dependency.Name,
			dependency.Version, license.Name)
	}

	if license.Restriction < licenseRestriction {
		return fmt.Errorf("Dependency '%s@%s' uses license '%s' which is not allowed on applications that run on customer machines.",
			dependency.Name, dependency.Version, license.Name)
	}
	return nil
}
