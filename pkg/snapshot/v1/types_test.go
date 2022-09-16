package snapshot_test

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	fileinfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, fileinfo := range fileinfos {
		name := fileinfo.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			contents, err := ioutil.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			var snapshot snapshotTypes.Snapshot
			if err := json.Unmarshal(contents, &snapshot); err != nil {
				t.Error(err)
			}
		})
	}
}
