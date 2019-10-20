package acmeclient

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/go-acme/lego/v3/registration"
	"github.com/gogo/protobuf/proto"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sClientDynamic "k8s.io/client-go/dynamic"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
)

type Controller struct {
	redisPool  *pool.Pool
	httpClient *http.Client
	snapshotCh <-chan watt.Snapshot

	secretsGetter k8sClientCoreV1.SecretsGetter
	hostsGetter   k8sClientDynamic.NamespaceableResourceInterface

	hosts   []watt.Host
	secrets []*k8sTypesCoreV1.Secret
}

func NewController(
	redisPool *pool.Pool,
	httpClient *http.Client,
	snapshotCh <-chan watt.Snapshot,
	secretsGetter k8sClientCoreV1.SecretsGetter,
	dynamicClient k8sClientDynamic.Interface,
) *Controller {
	return &Controller{
		redisPool:  redisPool,
		httpClient: httpClient,
		snapshotCh: snapshotCh,

		secretsGetter: secretsGetter,
		hostsGetter:   dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "hosts"}),
	}
}

func (c *Controller) Worker(logger types.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	for {
		select {
		case <-ticker.C:
			c.rectify(logger)
		case snapshot, ok := <-c.snapshotCh:
			if !ok {
				ticker.Stop()
				return
			}
			if c.processSnapshot(snapshot) {
				c.rectify(logger)
			}
		}
	}
}

func (c *Controller) processSnapshot(snapshot watt.Snapshot) (changed bool) {
	hosts := append([]watt.Host(nil), snapshot.Kubernetes.Host...)
	sort.SliceStable(hosts, func(i, j int) bool {
		switch {
		case hosts[i].GetNamespace() < hosts[j].GetNamespace():
			return true
		case hosts[i].GetNamespace() == hosts[j].GetNamespace():
			return hosts[i].GetName() < hosts[j].GetName()
		case hosts[i].GetNamespace() > hosts[j].GetNamespace():
			return false
		}
		panic("not reached")
	})
	if len(hosts) != len(c.hosts) {
		changed = true
	} else {
		for i := range hosts {
			if hosts[i].GetNamespace() != c.hosts[i].GetNamespace() {
				changed = true
				break
			}
			if hosts[i].GetName() != c.hosts[i].GetName() {
				changed = true
				break
			}
			if !proto.Equal(hosts[i].Spec, c.hosts[i].Spec) {
				changed = true
				break
			}
		}
	}

	secrets := append([]*k8sTypesCoreV1.Secret(nil), snapshot.Kubernetes.Secret...)
	sort.SliceStable(secrets, func(i, j int) bool {
		switch {
		case secrets[i].GetNamespace() < secrets[j].GetNamespace():
			return true
		case secrets[i].GetNamespace() == secrets[j].GetNamespace():
			return secrets[i].GetName() < secrets[j].GetName()
		case secrets[i].GetNamespace() > secrets[j].GetNamespace():
			return false
		}
		panic("not reached")
	})
	if len(secrets) != len(c.secrets) {
		changed = true
	} else {
		for i := range secrets {
			if !proto.Equal(secrets[i], c.secrets[i]) {
				changed = true
				break
			}
		}
	}

	if changed {
		c.hosts = hosts
		c.secrets = secrets
	}
	return changed
}

func (c *Controller) getSecret(namespace, name string) *k8sTypesCoreV1.Secret {
	for _, secret := range c.secrets {
		if secret.GetNamespace() == namespace && secret.GetName() == name {
			return secret
		}
	}
	return nil
}

func (c *Controller) updateHostSpec(namespace, name string, spec *ambassadorTypesV2.HostSpec) error {
	_, err := c.hostsGetter.Namespace(namespace).Update(&k8sTypesUnstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "Host",
			"metadata": map[string]string{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}, k8sTypesMetaV1.UpdateOptions{})
	return err
}

type providerKey struct {
	Authority            string
	Email                string
	PrivateKeySecretName string
}

func (c *Controller) rectify(logger types.Logger) {
	// tlsSecretXXX[namespace][tls_secret_name]
	tlsSecretProviders := make(map[string]map[string]*ambassadorTypesV2.ACMEProviderSpec)
	tlsSecretHostnames := make(map[string]map[string][]string)

	// Use 'c.hosts' and 'c.secrets' to populate
	// 'tlsSecretProviders' and 'tlsSecretHostnames'
	acmeProviders := make(map[providerKey]*ambassadorTypesV2.ACMEProviderSpec)
	for _, host := range c.hosts {
		spec := deepCopyHostSpec(host.Spec)

		fillDefaults(spec)
		if !proto.Equal(spec, host.Spec) {
			if err := c.updateHostSpec(host.GetNamespace(), host.GetName(), spec); err != nil {
				logger.Errorln(err)
			}
			continue
		}

		if spec.AcmeProvider.Authority == "none" {
			continue
		}

		if c.getSecret(host.GetNamespace(), spec.AcmeProvider.PrivateKeySecret.Name) == nil {
			err := createUserPrivateKey(c.secretsGetter, host.GetNamespace(), spec.AcmeProvider.PrivateKeySecret.Name)
			if err != nil {
				logger.Errorln(err)
			}
			continue
		}

		if spec.AcmeProvider.Registration == "" {
			hashKey := providerKey{
				Authority:            spec.AcmeProvider.Authority,
				Email:                spec.AcmeProvider.Email,
				PrivateKeySecretName: spec.AcmeProvider.PrivateKeySecret.Name,
			}
			if dup, hasDup := acmeProviders[hashKey]; !hasDup {
				err := c.userRegister(host.GetNamespace(), spec.AcmeProvider)
				if err != nil {
					logger.Errorln(err)
					continue
				}
			} else {
				spec.AcmeProvider = dup
			}
			if err := c.updateHostSpec(host.GetNamespace(), host.GetName(), spec); err != nil {
				logger.Errorln(err)
			}
			continue
		}

		// If we made it this far without "continue", then
		// we're ready to aquire a certificate for this Host.
		if _, nsSeen := tlsSecretProviders[host.GetNamespace()]; !nsSeen {
			tlsSecretProviders[host.GetNamespace()] = make(map[string]*ambassadorTypesV2.ACMEProviderSpec)
			tlsSecretHostnames[host.GetNamespace()] = make(map[string][]string)
		}
		if dup, hasDup := tlsSecretProviders[host.GetNamespace()][spec.TlsSecret.Name]; hasDup {
			if !proto.Equal(dup, spec.AcmeProvider) {
				logger.Errorln(errors.New("acmeProvider mismatch"))
			}
		} else {
			tlsSecretProviders[host.GetNamespace()][spec.TlsSecret.Name] = spec.AcmeProvider
		}
		tlsSecretHostnames[host.GetNamespace()][spec.TlsSecret.Name] = append(tlsSecretHostnames[host.GetNamespace()][spec.TlsSecret.Name], spec.Hostname)
	}

	// Now act on 'tlsSecretProviders' and 'tlsSecretHostnames'
	for namespace := range tlsSecretProviders {
		for tlsSecretName := range tlsSecretProviders[namespace] {
			needsRenew := false
			secret := c.getSecret(namespace, tlsSecretName)
			if secret == nil {
				// "renew" certs that we don't even have an old version of
				needsRenew = true
			} else {

				if cert, err := parseTLSSecret(secret); err != nil {
					// "renew" invalid certs
					needsRenew = true
				} else {
					// renew certs if they're >2/3 of the way through their lifecycle
					needsRenew = needsRenew || time.Now().After(cert.NotBefore.Add(2*cert.NotAfter.Sub(cert.NotBefore)/3))
					// or if the list of hostnames we want on it changed
					sort.Strings(tlsSecretHostnames[namespace][tlsSecretName])
					needsRenew = needsRenew || !stringSliceEqual(subjects(cert), tlsSecretHostnames[namespace][tlsSecretName])
				}
			}

			if needsRenew {
				acmeProvider := tlsSecretProviders[namespace][tlsSecretName]
				var user acmeUser
				var err error
				user.Email = acmeProvider.Email
				user.PrivateKey, err = parseUserPrivateKey(c.getSecret(namespace, acmeProvider.PrivateKeySecret.Name))
				if err != nil {
					logger.Errorln(err)
					continue
				}
				var reg registration.Resource
				if err = json.Unmarshal([]byte(acmeProvider.Registration), &reg); err != nil {
					logger.Errorln(err)
					continue
				}
				user.Registration = &reg

				certResource, err := obtainCertificate(
					c.httpClient,
					c.redisPool,
					acmeProvider.Authority,
					&user,
					tlsSecretHostnames[namespace][tlsSecretName])
				if err != nil {
					logger.Errorln(err)
					continue
				}
				if err = storeCertificate(c.secretsGetter, tlsSecretName, namespace, certResource); err != nil {
					logger.Errorln(err)
					continue
				}
			}
		}
	}
}

func fillDefaults(spec *ambassadorTypesV2.HostSpec) {
	if spec.Selector == nil {
		spec.Selector = &k8sTypesMetaV1.LabelSelector{}
	}
	if len(spec.Selector.MatchLabels)+len(spec.Selector.MatchExpressions) == 0 {
		spec.Selector.MatchLabels = map[string]string{
			"hostname": spec.Hostname,
		}
	}
	if spec.AcmeProvider == nil {
		spec.AcmeProvider = &ambassadorTypesV2.ACMEProviderSpec{}
	}
	if spec.AcmeProvider.Authority == "" {
		spec.AcmeProvider.Authority = "https://acme-staging-v02.api.letsencrypt.org/directory" // "https://acme-v02.api.letsencrypt.org/directory"
	}
	if spec.AcmeProvider.Authority != "none" {
		if spec.AcmeProvider.PrivateKeySecret == nil {
			spec.AcmeProvider.PrivateKeySecret = &k8sTypesCoreV1.LocalObjectReference{}
		}
		if spec.AcmeProvider.PrivateKeySecret.Name == "" {
			spec.AcmeProvider.PrivateKeySecret.Name = nameEncode(spec.AcmeProvider.Authority) + "--" + nameEncode(spec.AcmeProvider.Email)
		}
		if spec.TlsSecret == nil {
			spec.TlsSecret = &k8sTypesCoreV1.LocalObjectReference{}
		}
		if spec.TlsSecret.Name == "" {
			spec.TlsSecret.Name = nameEncode(spec.AcmeProvider.Authority) + "--" + nameEncode(spec.AcmeProvider.Email) + "--" + nameEncode(spec.AcmeProvider.PrivateKeySecret.Name)
		}
	}
}
