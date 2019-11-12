package watt_test

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/datawire/apro/cmd/amb-sidecar/watt"
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
			var snapshot watt.Snapshot
			if err := json.Unmarshal(contents, &snapshot); err != nil {
				t.Error(err)
			}
		})
	}
}
