package entrypoint_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

func AnySnapshot(_ *snapshot.Snapshot) bool {
	return true
}

func AnyConfig(_ *v3bootstrap.Bootstrap) bool {
	return true
}

func TestFake(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
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

func TestFakeIstioCert(t *testing.T) {
	// Don't ask for the EnvoyConfig yet, 'cause we don't use it.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: false}, nil)
	f.AutoFlush(true)

	f.UpsertFile("testdata/tls-snap.yaml")

	// fmt.Println(f.GetSnapshotString())

	k := f.GetSnapshot(AnySnapshot).Kubernetes

	if len(k.Secrets) != 1 {
		t.Errorf("needed 1 secret, got %d", len(k.Secrets))
	}

	istioSecret := kates.Secret{
		TypeMeta: kates.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "test-istio-secret",
			Namespace: "default",
		},
		Type: kates.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": []byte("not-real-cert"),
			"tls.crt": []byte("not-real-pem"),
		},
	}

	f.SendIstioCertUpdate(entrypoint.IstioCertUpdate{
		Op:        "update",
		Name:      "test-istio-secret",
		Namespace: "default",
		Secret:    &istioSecret,
	})

	k = f.GetSnapshot(AnySnapshot).Kubernetes

	fmt.Println(Jsonify(k))

	if len(k.Secrets) != 2 {
		t.Errorf("needed 2 secrets, got %d", len(k.Secrets))
	}
}
