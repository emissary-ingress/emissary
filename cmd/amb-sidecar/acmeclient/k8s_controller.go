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

	hosts   []*ambassadorTypesV2.Host
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
			logger.Debugln("processing snapshot change...")
			if c.processSnapshot(snapshot) {
				c.rectify(logger)
			}
		}
	}
}

func (c *Controller) processSnapshot(snapshot watt.Snapshot) (changed bool) {
	hosts := append([]*ambassadorTypesV2.Host(nil), snapshot.Kubernetes.Host...)
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
			if !hostsEqual(hosts[i], c.hosts[i]) {
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
			if !secretsEqual(secrets[i], c.secrets[i]) {
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

func (c *Controller) updateHost(host *ambassadorTypesV2.Host) error {
	_, err := c.hostsGetter.Namespace(host.GetNamespace()).Update(&k8sTypesUnstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "Host",
			"metadata":   unstructureMetadata(host.ObjectMeta),
			"spec":       host.Spec,
			"status":     host.Status,
		},
	}, k8sTypesMetaV1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "update %q.%q", host.GetName(), host.GetNamespace())
	}
	return err
}

func (c *Controller) recordHostPending(logger types.Logger, host *ambassadorTypesV2.Host, phaseCompleted, phasePending ambassadorTypesV2.HostPhase) {
	logger.Debugln("updating pending host")
	host.Status.State = ambassadorTypesV2.HostState_Pending
	host.Status.PhaseCompleted = phaseCompleted
	host.Status.PhasePending = phasePending
	host.Status.Reason = ""
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
	}
}

func (c *Controller) recordHostReady(logger types.Logger, host *ambassadorTypesV2.Host) {
	logger.Debugln("updating ready host")
	host.Status.State = ambassadorTypesV2.HostState_Ready
	host.Status.PhaseCompleted = ambassadorTypesV2.HostPhase_NA
	host.Status.PhasePending = ambassadorTypesV2.HostPhase_NA
	host.Status.Reason = ""
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
	}
}

func (c *Controller) recordHostError(logger types.Logger, host *ambassadorTypesV2.Host, phase ambassadorTypesV2.HostPhase, err error) {
	logger.Debugln("updating errored host:", err)
	host.Status.State = ambassadorTypesV2.HostState_Error
	host.Status.PhasePending = phase
	host.Status.Reason = err.Error()
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
	}
}

type providerKey struct {
	Authority            string
	Email                string
	PrivateKeySecretName string
}

func (c *Controller) rectify(logger types.Logger) {
	logger.Debugln("rectify...")
	// tlsSecretXXX[namespace][tls_secret_name]
	tlsSecretProviders := make(map[string]map[string]*ambassadorTypesV2.ACMEProviderSpec)
	tlsSecretHostnames := make(map[string]map[string][]string)

	// Use 'c.hosts' and 'c.secrets' to populate
	// 'tlsSecretProviders' and 'tlsSecretHostnames'
	acmeProviders := make(map[providerKey]*ambassadorTypesV2.ACMEProviderSpec)
	for _, _host := range c.hosts {
		host := deepCopyHost(_host)
		logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
		logger.Debugln("processing host...")

		FillDefaults(host)
		if !proto.Equal(host.Spec, _host.Spec) {
			logger.Debugln("saving defaults")
			c.recordHostPending(logger, host,
				ambassadorTypesV2.HostPhase_DefaultsFilled,
				ambassadorTypesV2.HostPhase_NA)
			continue
		}

		switch host.Status.TlsCertificateSource {
		case ambassadorTypesV2.HostTLSCertificateSource_None:
			logger.Debugln("Host does not use TLS")
			c.recordHostReady(logger, host)
			continue
		case ambassadorTypesV2.HostTLSCertificateSource_Other:
			logger.Debugln("Host uses externally provisioned TLS certificate")
			if c.getSecret(host.GetMetadata().GetNamespace(), host.Spec.TlsSecret.Name) == nil {
				c.recordHostError(logger, host,
					ambassadorTypesV2.HostPhase_NA,
					errors.New("tlsSecret does not exist"))
			} else {
				// TODO: Maybe validate that the secret contents are valid?
				c.recordHostReady(logger, host)
			}
			continue
		case ambassadorTypesV2.HostTLSCertificateSource_ACME:
			// proceed
		}

		if c.getSecret(host.GetNamespace(), host.Spec.AcmeProvider.PrivateKeySecret.Name) == nil {
			logger.Debugln("creating user private key")
			err := createUserPrivateKey(c.secretsGetter, host.GetNamespace(), host.Spec.AcmeProvider.PrivateKeySecret.Name)
			if err != nil {
				c.recordHostError(logger, host,
					ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
					err)
			} else {
				c.recordHostPending(logger, host,
					ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
					ambassadorTypesV2.HostPhase_ACMEUserRegistered)
			}
			continue
		}

		if host.Spec.AcmeProvider.Registration == "" {
			logger.Debugln("registering user")
			hashKey := providerKey{
				Authority:            host.Spec.AcmeProvider.Authority,
				Email:                host.Spec.AcmeProvider.Email,
				PrivateKeySecretName: host.Spec.AcmeProvider.PrivateKeySecret.Name,
			}
			if dup, hasDup := acmeProviders[hashKey]; !hasDup {
				err := c.userRegister(host.GetNamespace(), host.Spec.AcmeProvider)
				if err != nil {
					c.recordHostError(logger, host,
						ambassadorTypesV2.HostPhase_ACMEUserRegistered,
						err)
					continue
				}
			} else {
				host.Spec.AcmeProvider = dup
				c.recordHostPending(logger, host,
					ambassadorTypesV2.HostPhase_ACMEUserRegistered,
					ambassadorTypesV2.HostPhase_ACMECertificateChallenge)
			}
			continue
		}

		// If we made it this far without "continue", then
		// we're ready to aquire a certificate for this Host.
		logger.Debugln("queuing for certificate check")
		if _, nsSeen := tlsSecretProviders[host.GetNamespace()]; !nsSeen {
			tlsSecretProviders[host.GetNamespace()] = make(map[string]*ambassadorTypesV2.ACMEProviderSpec)
			tlsSecretHostnames[host.GetNamespace()] = make(map[string][]string)
		}
		if dup, hasDup := tlsSecretProviders[host.GetNamespace()][host.Spec.TlsSecret.Name]; hasDup {
			if !proto.Equal(dup, host.Spec.AcmeProvider) {
				// TODO: record this
				logger.Errorln(errors.New("acmeProvider mismatch"))
			}
		} else {
			tlsSecretProviders[host.GetNamespace()][host.Spec.TlsSecret.Name] = host.Spec.AcmeProvider
		}
		tlsSecretHostnames[host.GetNamespace()][host.Spec.TlsSecret.Name] = append(tlsSecretHostnames[host.GetNamespace()][host.Spec.TlsSecret.Name], host.Spec.Hostname)
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
					// TODO: record this for all relevant hosts
					logger.Errorln(err)
					continue
				}
				var reg registration.Resource
				if err = json.Unmarshal([]byte(acmeProvider.Registration), &reg); err != nil {
					// TODO: record this for all relevant hosts
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
					// TODO: record this for all relevant hosts
					logger.Errorln(err)
					continue
				}
				if err = storeCertificate(c.secretsGetter, tlsSecretName, namespace, certResource); err != nil {
					// TODO: record this for all relevant hosts
					logger.Errorln(err)
					continue
				}
			}
			// TODO: record success for all relevant hosts
		}
	}
}

func FillDefaults(host *ambassadorTypesV2.Host) {
	if host.Spec == nil {
		host.Spec = &ambassadorTypesV2.HostSpec{}
	}
	if host.Spec.Selector == nil {
		host.Spec.Selector = &k8sTypesMetaV1.LabelSelector{}
	}
	if len(host.Spec.Selector.MatchLabels)+len(host.Spec.Selector.MatchExpressions) == 0 {
		host.Spec.Selector.MatchLabels = map[string]string{
			"hostname": host.Spec.Hostname,
		}
	}
	if host.Spec.AcmeProvider == nil {
		host.Spec.AcmeProvider = &ambassadorTypesV2.ACMEProviderSpec{}
	}
	if host.Spec.AcmeProvider.Authority == "" {
		host.Spec.AcmeProvider.Authority = "https://acme-staging-v02.api.letsencrypt.org/directory" // "https://acme-v02.api.letsencrypt.org/directory"
	}
	if host.Spec.AcmeProvider.Authority != "none" {
		if host.Spec.AcmeProvider.PrivateKeySecret == nil {
			host.Spec.AcmeProvider.PrivateKeySecret = &k8sTypesCoreV1.LocalObjectReference{}
		}
		if host.Spec.AcmeProvider.PrivateKeySecret.Name == "" {
			host.Spec.AcmeProvider.PrivateKeySecret.Name = NameEncode(host.Spec.AcmeProvider.Authority) + "--" + NameEncode(host.Spec.AcmeProvider.Email)
		}
		if host.Spec.TlsSecret == nil {
			host.Spec.TlsSecret = &k8sTypesCoreV1.LocalObjectReference{}
		}
		if host.Spec.TlsSecret.Name == "" {
			host.Spec.TlsSecret.Name = NameEncode(host.Spec.AcmeProvider.Authority) + "--" + NameEncode(host.Spec.AcmeProvider.Email) + "--" + NameEncode(host.Spec.AcmeProvider.PrivateKeySecret.Name)
		}
	}
	if host.Status == nil {
		host.Status = &ambassadorTypesV2.HostStatus{}
	}
	if host.Spec.AcmeProvider.Authority != "none" {
		host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_ACME
	} else if host.Spec.TlsSecret.Name == "" {
		host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_Other
	} else {
		host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_None
	}
}
