package entrypoint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kates/k8sresourcetypes"
	"github.com/datawire/dlib/dlog"
)

var serviceTests = []struct {
	Module            moduleResolver
	Service           string
	Namespace         string
	ExpectedService   string
	ExpectedNamespace string
	ExpectedPort      string
}{
	{
		moduleResolver{},
		"service-name.namespace:3000",
		"other-namespace",
		"service-name",
		"namespace",
		"3000",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"service-name.namespace:3000",
		"other-namespace",
		"service-name",
		"namespace",
		"3000",
	},
	{
		moduleResolver{},
		"service-name.namespace",
		"other-namespace",
		"service-name",
		"namespace",
		"",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"service-name.namespace",
		"other-namespace",
		"service-name",
		"namespace",
		"",
	},
	{
		moduleResolver{},
		"service-name.namespace.svc.cluster.local:3000",
		"other-namespace",
		"service-name",
		"namespace",
		"3000",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"service-name.namespace.svc.cluster.local:3000",
		"other-namespace",
		"service-name",
		"namespace",
		"3000",
	},
	{
		moduleResolver{},
		"service-name.namespace.svc.cluster.local",
		"other-namespace",
		"service-name",
		"namespace",
		"",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"service-name.namespace.svc.cluster.local",
		"other-namespace",
		"service-name",
		"namespace",
		"",
	},
	{
		moduleResolver{},
		"service-name:3000",
		"other-namespace",
		"service-name",
		"other-namespace",
		"3000",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"service-name:3000",
		"other-namespace",
		"service-name",
		"default",
		"3000",
	},
	{
		moduleResolver{},
		"service-name",
		"other-namespace",
		"service-name",
		"other-namespace",
		"",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"service-name",
		"other-namespace",
		"service-name",
		"default",
		"",
	},
	{
		moduleResolver{},
		"1.2.3.4",
		"other-namespace",
		"1.2.3.4",
		"other-namespace",
		"",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"1.2.3.4:8080",
		"other-namespace",
		"1.2.3.4",
		"default",
		"8080",
	},
	{
		moduleResolver{},
		"1.2.3.4:blah",
		"other-namespace",
		"1",
		"2",
		"",
	},
	{
		moduleResolver{UseAmbassadorNamespaceForServiceResolution: true},
		"1.2.3.4:blah",
		"other-namespace",
		"1",
		"2",
		"",
	},
}

func TestParseService(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	cm := &k8sresourcetypes.ConfigMap{ObjectMeta: kates.ObjectMeta{Name: "foo", Namespace: "bar"}}
	for _, test := range serviceTests {
		// Make sure we ignore these also.
		for _, prefix := range []string{"", "http://", "https://"} {
			service := fmt.Sprintf("%s%s", prefix, test.Service)
			t.Run(fmt.Sprintf("%s,%s,%v", service, test.Namespace, test.Module.UseAmbassadorNamespaceForServiceResolution), func(t *testing.T) {
				name, namespace, port := test.Module.parseService(ctx, cm, service, test.Namespace)
				assert.Equal(t, test.ExpectedService, name)
				assert.Equal(t, test.ExpectedNamespace, namespace)
				assert.Equal(t, test.ExpectedPort, port)
			})
		}
	}
}
