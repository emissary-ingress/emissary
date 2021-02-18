package entrypoint_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/datawire/ambassador/cmd/entrypoint"
)

func TestFake(t *testing.T) {
	ctx := context.Background()
	f := entrypoint.RunFake(ctx)
	time.Sleep(1 * time.Second)
	f.ApplyFile("testdata/snapshot.yaml")
	fmt.Println(f.GetSnapshotString())

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
