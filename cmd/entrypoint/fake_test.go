package entrypoint_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
)

func AnySnapshot(_ *snapshot.Snapshot) bool {
	return true
}

func AnyConfig(_ *bootstrap.Bootstrap) bool {
	return true
}

func TestFake(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true})
	f.UpsertFile("testdata/snapshot.yaml")
	f.AutoFlush(true)
	fmt.Println(Jsonify(f.GetSnapshot(AnySnapshot)))
	fmt.Println(Jsonify(f.GetEnvoyConfig(AnyConfig)))

	f.Delete("Mapping", "default", "foo")

	fmt.Println(Jsonify(f.GetSnapshot(AnySnapshot)))
	fmt.Println(Jsonify(f.GetEnvoyConfig(AnyConfig)))
	/*f.ConsulEndpoints(endpointsBlob)
	f.ApplyFile()
	f.ApplyResources()
	f.Snapshot(snapshot1)
	f.Snapshot(snapshot2)
	f.Snapshot(snapshot3)
	f.Delete(namespace, name)
	f.Upsert(katesObject)
	f.UpsertString("kind: blah")*/

	// bluescape: create 50 hosts in different namespaces vs 50 hosts in the same namespace
	// consul data center other than dc1

}

func Jsonify(obj interface{}) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
