package detectlicense

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ReadPackageLicensesFromFile reads package licenses from a file.
func ReadPackageLicensesFromFile(name string) (map[string]map[License]struct{}, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}
	var pns map[string][]string
	if err = yaml.Unmarshal(data, &pns); err != nil {
		return nil, err
	}
	plm := make(map[string]map[License]struct{}, len(pns))
	for pkg, ids := range pns {
		lm := make(map[License]struct{}, len(ids))
		for _, id := range ids {
			l, ok := SpdxIdentifiers[id]
			if !ok {
				return nil, fmt.Errorf("%q is not a valid SPDX License identifier. See https://spdx.org/licenses/ for a full litst", id)
			}
			lm[l] = struct{}{}
		}
		plm[pkg] = lm
	}
	return plm, nil
}
