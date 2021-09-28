package detectlicense_test

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/datawire/ambassador/v2/pkg/mkopensource/detectlicense"
)

func licenseListEqual(a, b map[detectlicense.License]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for lic := range a {
		if _, ok := b[lic]; !ok {
			return false
		}
	}
	return true
}

func fmtLicenses(set map[detectlicense.License]struct{}) string {
	if len(set) == 0 {
		return "(none)"
	}
	list := make([]string, 0, len(set))
	for lic := range set {
		list = append(list, lic.Name)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

func TestIdentifyLicenses(t *testing.T) {
	// Mapping of `testdata/{NAME}/` names to detectlicense constants
	allLicenses := map[string]detectlicense.License{
		"Apache2":      detectlicense.Apache2,
		"BSD1":         detectlicense.BSD1,
		"BSD2":         detectlicense.BSD2,
		"BSD3":         detectlicense.BSD3,
		"ISC":          detectlicense.ISC,
		"MIT":          detectlicense.MIT,
		"MPL2":         detectlicense.MPL2,
		"CC-BY-SA-4.0": detectlicense.CcBySa40,
	}
	dirInfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, dirInfo := range dirInfos {
		if !dirInfo.IsDir() {
			continue
		}
		dirName := dirInfo.Name()
		t.Run(dirName, func(t *testing.T) {
			dirLicenses := make(map[detectlicense.License]struct{})
			for _, licName := range strings.Split(dirName, "+") {
				dirLicenses[allLicenses[licName]] = struct{}{}
			}

			fileInfos, err := ioutil.ReadDir(filepath.Join("testdata", dirName))
			if err != nil {
				t.Fatal(err)
			}
			for _, fileInfo := range fileInfos {
				fileName := fileInfo.Name()
				t.Run(fileName, func(t *testing.T) {
					fileBody, err := ioutil.ReadFile(filepath.Join("testdata", dirName, fileName))
					if err != nil {
						t.Fatal(err)
					}
					fileLicenses := detectlicense.IdentifyLicenses(fileBody)
					if !licenseListEqual(fileLicenses, dirLicenses) {
						t.Errorf("wrong result:\n"+
							"expected: %s\n"+
							"received: %s\n",
							fmtLicenses(dirLicenses),
							fmtLicenses(fileLicenses))
					}
				})
			}
		})
	}
}
