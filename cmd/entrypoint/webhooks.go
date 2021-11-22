package entrypoint

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

func handleWebhooks(ctx context.Context) error {
	// Create the webhook server
	scheme := runtime.NewScheme()
	utilruntime.Must(v2.AddToScheme(scheme))
	utilruntime.Must(v3alpha1.AddToScheme(scheme))
	webhook := &conversion.Webhook{}
	if err := webhook.InjectScheme(scheme); err != nil {
		return err
	}
	return nil
}
