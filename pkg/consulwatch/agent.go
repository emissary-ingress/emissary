package consulwatch

type Agent struct {
	// AmbassadorID is the ID of the Ambassador instance.
	AmbassadorID string

	// The Agent registers a Consul Service when it starts and then fetches the leaf TLS certificate from the Consul
	// HTTP API with this name.
	ConsulServiceName string

	// SecretNamespace is the Namespace where the TLS secret is managed.
	SecretNamespace string

	// SecretName is the Name of the TLS secret managed by this Agent.
	SecretName string
}

func NewAgent(spec ConsulWatchSpec) *Agent {
	ambassadorID := spec.Id
	consulServiceName := defConsulServiceName

	// get the secret namespace/name where
	secretNamespace, secretName := getNamespaceAndName(spec.Secret)

	// The secret name is the full name of the Kubernetes Secret that contains the TLS certificate provided
	// by Consul. If this value is set then the value of AMBASSADOR_ID is ignored when the name of the TLS secret is
	// computed.
	if ambassadorID != "" {
		consulServiceName += "-" + ambassadorID
	}

	if secretName == "" {
		secretName = consulServiceName + "-consul-connect"
	}

	return &Agent{
		AmbassadorID:      consulServiceName,
		SecretNamespace:   secretNamespace,
		SecretName:        secretName,
		ConsulServiceName: consulServiceName,
	}
}
