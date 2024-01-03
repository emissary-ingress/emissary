package defaults

import "time"

const (
	// APIExtNamespace is the default namespace where the APIExt server is installed
	APIExtNamespace = "emissary-system"

	// APIExtNamespace is the default name used for the ApiEXT server deployment
	APIEXTDeploymentName = "emissary-apiext"

	// ResyncPeriod is set to 0 so that by default it only re-syncs on changes rather than periodically
	ResyncPeriod = 0 * time.Second

	// RequeueAfter is the time to wait before a controller requeues the event for reconciliation
	RequeueAfter = 10 * time.Second

	// SubjectOrganization is the default organization used when generating certificates
	SubjectOrganization = "Ambassador Labs"

	// WebhookCASecretName is the default name where the root ca is stored
	WebhookCASecretName = "emissary-ingress-webhook-ca"

	// WebhookCASecretNamespace is the default namespace where the root ca is stored
	WebhookCASecretNamespace = "emissary-system"
)
