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
	logger.Debugf("updating pending host %d→%d", host.Status.PhaseCompleted, phaseCompleted)
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

// providerKey is used as a hash key to bucket Hosts by ACME account.
type providerKey struct {
	Authority            string
	Email                string
	PrivateKeySecretName string
}

func (c *Controller) rectify(logger types.Logger) {
	logger.Debugln("rectify: starting")

	// Phase 0→1[→2]: NA→DefaultsFilled[→ACMEUserPrivateKeyCreated]
	logger.Debugln("rectify: Phase 0→1[→2]: NA→DefaultsFilled[→ACMEUserPrivateKeyCreated]")
	// Record in 'acmeHosts' a list of Hosts that are ready for the next ACME phase
	var acmeHosts []*ambassadorTypesV2.Host
	for _, _host := range c.hosts {
		host := deepCopyHost(_host)
		logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
		logger.Debugln("rectify: processing host...")

		FillDefaults(host)
		if !proto.Equal(host.Spec, _host.Spec) {
			logger.Debugln("saving defaults")
			nextPhase := ambassadorTypesV2.HostPhase_NA
			if host.Status.TlsCertificateSource == ambassadorTypesV2.HostTLSCertificateSource_ACME {
				nextPhase = ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated
			}
			c.recordHostPending(logger, host,
				ambassadorTypesV2.HostPhase_DefaultsFilled,
				nextPhase)
			continue
		}

		switch host.Status.TlsCertificateSource {
		case ambassadorTypesV2.HostTLSCertificateSource_None:
			logger.Debugln("rectify: Host does not use TLS")
			c.recordHostReady(logger, host)
		case ambassadorTypesV2.HostTLSCertificateSource_Other:
			logger.Debugln("rectify: Host uses externally provisioned TLS certificate")
			if c.getSecret(host.GetNamespace(), host.Spec.TlsSecret.Name) == nil {
				c.recordHostError(logger, host,
					ambassadorTypesV2.HostPhase_NA,
					errors.New("tlsSecret does not exist"))
			} else {
				// TODO: Maybe validate that the secret contents are valid?
				c.recordHostReady(logger, host)
			}
		case ambassadorTypesV2.HostTLSCertificateSource_ACME:
			if c.getSecret(host.GetNamespace(), host.Spec.AcmeProvider.PrivateKeySecret.Name) == nil {
				logger.Debugln("rectify: creating user private key")
				err := createUserPrivateKey(c.secretsGetter, host.GetNamespace(), host.Spec.AcmeProvider.PrivateKeySecret.Name)
				if err != nil {
					c.recordHostError(logger, host,
						ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
						err)
				}
			} else if host.Status.State == ambassadorTypesV2.HostState_Pending && host.Status.PhaseCompleted < ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated {
				c.recordHostPending(logger, host,
					ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
					ambassadorTypesV2.HostPhase_ACMEUserRegistered)
			} else {
				logger.Debugln("rectify: accepting host for next phase")
				acmeHosts = append(acmeHosts, host)
			}
		}
	}

	// Phase 2→3: ACMEUserPrivateKeyCreated→ACMEUserRegistered
	// Populate 'acmeHostsBySecret' and 'acmeProviderBySecret' from 'acmeHosts'.
	logger.Debugln("rectify: Phase 2→3: ACMEUserPrivateKeyCreated→ACMEUserRegistered")
	acmeHostsBySecret := make(map[string]map[string][]*ambassadorTypesV2.Host)
	for _, host := range acmeHosts {
		if _, nsSeen := acmeHostsBySecret[host.GetNamespace()]; !nsSeen {
			acmeHostsBySecret[host.GetNamespace()] = make(map[string][]*ambassadorTypesV2.Host)
		}
		acmeHostsBySecret[host.GetNamespace()][host.Spec.TlsSecret.Name] = append(acmeHostsBySecret[host.GetNamespace()][host.Spec.TlsSecret.Name],
			host)
	}
	acmeProviderBySecret := make(map[string]map[string]*ambassadorTypesV2.ACMEProviderSpec)
	for namespace := range acmeHostsBySecret {
		logger := logger.WithField("namespace", namespace)
		acmeProviderBySecret[namespace] = make(map[string]*ambassadorTypesV2.ACMEProviderSpec)
		for tlsSecretName := range acmeHostsBySecret[namespace] {
			logger := logger.WithField("secret", tlsSecretName)
			logger.Debugf("rectify: processing hosts that share secret=%q: %v",
				tlsSecretName,
				func() []string {
					hosts := make([]string, 0, len(acmeHostsBySecret[namespace][tlsSecretName]))
					for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
						hosts = append(hosts, host.GetName())
					}
					sort.Strings(hosts)
					return hosts
				}())
			hostsByProvider := make(map[providerKey][]*ambassadorTypesV2.Host)
			for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
				hashKey := providerKey{
					Authority:            host.Spec.AcmeProvider.Authority,
					Email:                host.Spec.AcmeProvider.Email,
					PrivateKeySecretName: host.Spec.AcmeProvider.PrivateKeySecret.Name,
				}
				hostsByProvider[hashKey] = append(hostsByProvider[hashKey], host)
			}
			dirty := false
			for _, hosts := range hostsByProvider {
				registration := ""
				for _, host := range hosts {
					if host.Spec.AcmeProvider.Registration != "" {
						if registration != "" && registration != host.Spec.AcmeProvider.Registration {
							// TODO: Report this warning to the user, not just on stderr
							logger.Warningf("host=%q has disagreeing ACME registration: %q",
								host.GetName(), host.Spec.AcmeProvider.Registration)
						}
						registration = host.Spec.AcmeProvider.Registration
					}
				}
				if registration != "" {
					logger.Debugln("found existing registration")
				} else {
					logger.Debugln("registering ACME user...")
					var err error
					registration, err = c.userRegister(namespace, hosts[0].Spec.AcmeProvider)
					if err != nil {
						for _, host := range hosts {
							logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
							c.recordHostError(logger, host,
								ambassadorTypesV2.HostPhase_ACMEUserRegistered,
								err)
						}
						dirty = true
						continue
					}
				}
				for _, host := range hosts {
					logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
					if host.Spec.AcmeProvider.Registration != registration {
						host.Spec.AcmeProvider.Registration = registration
						c.recordHostPending(logger, host,
							ambassadorTypesV2.HostPhase_ACMEUserRegistered,
							ambassadorTypesV2.HostPhase_ACMECertificateChallenge)
						dirty = true
					}
				}
			}
			if dirty {
				logger.Debugln("1 or more hosts changed, ignoring secret until next snapshot...")
				continue
			}

			providerKeys := make([]providerKey, 0, len(hostsByProvider))
			for key := range hostsByProvider {
				providerKeys = append(providerKeys, key)
			}
			if len(providerKeys) > 1 {
				sort.Slice(providerKeys, func(i, j int) bool {
					// return 'true' if we'd pick 'providerKeys[i]' over 'providerKeys[j]'
					switch {
					// choose the one with the most hosts
					case len(hostsByProvider[providerKeys[i]]) > len(hostsByProvider[providerKeys[j]]):
						return true
					case len(hostsByProvider[providerKeys[i]]) < len(hostsByProvider[providerKeys[j]]):
						return false
					// as a tie-breaker, choose based on authority lexicographic sorting
					case providerKeys[i].Authority < providerKeys[j].Authority:
						return true
					case providerKeys[i].Authority > providerKeys[j].Authority:
						return false
					// as a 2nd tie-breaker, choose based on email lexicographic sorting
					case providerKeys[i].Email < providerKeys[j].Email:
						return true
					case providerKeys[i].Email > providerKeys[j].Email:
						return false
					// as a final tie-breaker, choose based on private key secret name lexicographic sorting
					default:
						return providerKeys[i].PrivateKeySecretName < providerKeys[j].PrivateKeySecretName
					}
				})
				// TODO: Report this warning to the user, not just on stderr
				logger.Warningln("there were multiple ACME providers specified for this secret")
			}
			logger.Debugln("rectify: accepting secret for next phase")
			acmeProviderBySecret[namespace][tlsSecretName] = hostsByProvider[providerKeys[0]][0].Spec.AcmeProvider
		}
	}

	// Phase 3→4: ACMEUserRegistered→ACMECertificateChallenge
	// Now act on 'acmeProviderBySecret' and 'acmeHostsBySecret'
	logger.Debugln("rectify: Phase 3→4: ACMEUserRegistered→ACMECertificateChallenge")
	for namespace := range acmeProviderBySecret {
		logger := logger.WithField("namespace", namespace)
		for tlsSecretName := range acmeProviderBySecret[namespace] {
			logger := logger.WithField("secret", tlsSecretName)
			hostnames := make([]string, 0, len(acmeHostsBySecret[namespace][tlsSecretName]))
			for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
				hostnames = append(hostnames, host.Spec.Hostname)
			}
			sort.Strings(hostnames)
			logger.Debugf("rectify: processing secret=%q (hostnames=%v)", tlsSecretName, hostnames)

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
					needsRenew = needsRenew || !stringSliceEqual(subjects(cert), hostnames)
				}
			}

			logger.Debugf("rectify: needsRenew=%v", needsRenew)
			if needsRenew {
				acmeProvider := acmeProviderBySecret[namespace][tlsSecretName]
				var user acmeUser
				var err error
				user.Email = acmeProvider.Email
				user.PrivateKey, err = parseUserPrivateKey(c.getSecret(namespace, acmeProvider.PrivateKeySecret.Name))
				if err != nil {
					for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
						logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
						c.recordHostError(logger, host,
							ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
							err)
					}
					continue
				}
				var reg registration.Resource
				if err = json.Unmarshal([]byte(acmeProvider.Registration), &reg); err != nil {
					for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
						logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
						c.recordHostError(logger, host,
							ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
							err)
					}
					continue
				}
				user.Registration = &reg

				logger.Debugln("rectify: requesting certificate...")
				certResource, err := obtainCertificate(
					c.httpClient,
					c.redisPool,
					acmeProvider.Authority,
					&user,
					hostnames)
				if err != nil {
					for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
						logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
						c.recordHostError(logger, host,
							ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
							err)
					}
					continue
				}
				if err = storeCertificate(c.secretsGetter, tlsSecretName, namespace, certResource); err != nil {
					for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
						logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
						c.recordHostError(logger, host,
							ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
							err)
					}
					continue
				}
			}
			logger.Debugln("rectify: updating host status(s)")
			for _, host := range acmeHostsBySecret[namespace][tlsSecretName] {
				if host.Status.State != ambassadorTypesV2.HostState_Ready {
					logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
					// There's a small-but-possible chance that a previous run already set this, but
					// we're operating on an outdated snapshot that has the `storeCertificate()`
					// call but not the `c.recordHostReady()` call.  We might get a 409 Conflict,
					// but that's OK; there's no actual information that can get lost here (unlike
					// when we're filling host.Spec.AcmeProvider.Registration).
					c.recordHostReady(logger, host)
				}
			}
		}
	}
	logger.Debugln("rectify: finished")
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
		host.Spec.AcmeProvider.Authority = "https://acme-v02.api.letsencrypt.org/directory"
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
