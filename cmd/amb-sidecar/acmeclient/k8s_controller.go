package acmeclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/go-acme/lego/v3/registration"
	"github.com/gogo/protobuf/proto"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sClientDynamic "k8s.io/client-go/dynamic"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/apro/cmd/amb-sidecar/events"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
)

type Controller struct {
	redisPool  *pool.Pool
	httpClient *http.Client

	snapshotCh  <-chan watt.Snapshot
	eventLogger *events.EventLogger

	secretsGetter k8sClientCoreV1.SecretsGetter
	hostsGetter   k8sClientDynamic.NamespaceableResourceInterface

	hosts   []*ambassadorTypesV2.Host
	secrets []*k8sTypesCoreV1.Secret
}

func NewController(
	redisPool *pool.Pool,
	httpClient *http.Client,
	snapshotCh <-chan watt.Snapshot,
	eventLogger *events.EventLogger,
	secretsGetter k8sClientCoreV1.SecretsGetter,
	dynamicClient k8sClientDynamic.Interface,
) *Controller {
	return &Controller{
		redisPool:   redisPool,
		httpClient:  httpClient,
		snapshotCh:  snapshotCh,
		eventLogger: eventLogger,

		secretsGetter: secretsGetter,
		hostsGetter:   dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "hosts"}),
	}
}

func (c *Controller) Worker(logger dlog.Logger) {
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
	_, err := c.hostsGetter.Namespace(host.GetNamespace()).Update(unstructureHost(host), k8sTypesMetaV1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "update %q.%q", host.GetName(), host.GetNamespace())
	}
	return err
}

func (c *Controller) recordHostPending(logger dlog.Logger, host *ambassadorTypesV2.Host, phaseCompleted, phasePending ambassadorTypesV2.HostPhase, reasonPending string) {
	logger.Debugf("updating pending host %d→%d", host.Status.PhaseCompleted, phaseCompleted)
	if phaseCompleted <= host.Status.PhaseCompleted {
		logger.Debugf("^^ THIS IS A BUG ^^: %d→%d is not a progression", host.Status.PhaseCompleted, phaseCompleted)
	}
	host.Status.State = ambassadorTypesV2.HostState_Pending
	host.Status.PhaseCompleted = phaseCompleted
	host.Status.PhasePending = phasePending
	host.Status.Reason = ""
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
	}
	c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeNormal, "Pending", reasonPending)
}

func (c *Controller) recordHostReady(logger dlog.Logger, host *ambassadorTypesV2.Host, readyReason string) {
	logger.Debugln("updating ready host")
	host.Status.State = ambassadorTypesV2.HostState_Ready
	host.Status.PhaseCompleted = ambassadorTypesV2.HostPhase_NA
	host.Status.PhasePending = ambassadorTypesV2.HostPhase_NA
	host.Status.Reason = ""
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
	}
	c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeNormal, "Ready", readyReason)
}

func (c *Controller) recordHostError(logger dlog.Logger, host *ambassadorTypesV2.Host, phase ambassadorTypesV2.HostPhase, err error) {
	logger.Debugln("updating errored host:", err)
	host.Status.State = ambassadorTypesV2.HostState_Error
	host.Status.PhasePending = phase
	host.Status.Reason = err.Error()
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
	}
	c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeWarning, "Error", err.Error())
}

func (c *Controller) recordHostsError(logger dlog.Logger, hosts []*ambassadorTypesV2.Host, phase ambassadorTypesV2.HostPhase, err error) {
	for _, host := range hosts {
		logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
		c.recordHostError(logger, host, phase, err)
	}
}

func (c *Controller) recordHostsEvent(hosts []*ambassadorTypesV2.Host, reason string) {
	for _, host := range hosts {
		c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeNormal, "Pending", reason)
	}
}

// providerKey is used as a hash key to bucket Hosts by ACME account.
type providerKey struct {
	Authority            string
	Email                string
	PrivateKeySecretName string
}

func (c *Controller) rectify(logger dlog.Logger) {
	logger.Debugln("rectify: starting")

	acmeHosts := c.rectifyPhase1(logger)
	acmeHosts = c.rectifyPhase2(logger, acmeHosts)
	acmeHostsByTLSSecret, acmeProviderByTLSSecret := c.rectifyPhase3(logger, acmeHosts)
	c.rectifyPhase4(logger, acmeHostsByTLSSecret, acmeProviderByTLSSecret)
}

// Phase 0→1 (Pre-ACME): NA(state=Initial)→DefaultsFilled
func (c *Controller) rectifyPhase1(logger dlog.Logger) []*ambassadorTypesV2.Host {
	var nextPhase []*ambassadorTypesV2.Host

	logger.Debugln("rectify: Phase 0→1 (Pre-ACME): NA(state=Initial)→DefaultsFilled")
	for _, _host := range c.hosts {
		host := deepCopyHost(_host)
		logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
		logger.Debugln("rectify: processing Host...")

		FillDefaults(host)
		if !proto.Equal(host.Spec, _host.Spec) {
			logger.Debugln("rectify: Host: saving defaults")
			nextPhase := ambassadorTypesV2.HostPhase_NA
			if host.Status.TlsCertificateSource == ambassadorTypesV2.HostTLSCertificateSource_ACME {
				nextPhase = ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated
			}
			c.recordHostPending(logger, host,
				ambassadorTypesV2.HostPhase_DefaultsFilled,
				nextPhase, "waiting for Host DefaultsFilled change to be reflected in snapshot")
			continue
		}

		switch host.Status.TlsCertificateSource {
		case ambassadorTypesV2.HostTLSCertificateSource_None:
			logger.Debugln("rectify: Host: does not use TLS")
			c.recordHostReady(logger, host, "non-TLS Host marked ready")
		case ambassadorTypesV2.HostTLSCertificateSource_Other:
			logger.Debugln("rectify: Host: uses externally-provisioned TLS certificate")
			if c.getSecret(host.GetNamespace(), host.Spec.TlsSecret.Name) == nil {
				c.recordHostError(logger, host,
					ambassadorTypesV2.HostPhase_NA,
					errors.New("tlsSecret does not exist"))
			} else {
				// TODO: Maybe validate that the secret contents are valid?
				c.recordHostReady(logger, host, "Host with externally-provisioned TLS certificate marked Ready")
			}
		case ambassadorTypesV2.HostTLSCertificateSource_ACME:
			logger.Debugln("rectify: Host: accepting Host for next phase")
			nextPhase = append(nextPhase, host)
		default:
			logger.Debugf("rectify: Host: THIS IS A BUG: Unknown TlsCertificateSource", host.Status.TlsCertificateSource)
		}
	}

	return nextPhase
}

// Phase 1→2 (ACME account pre-registration): DefaultsFilled→ACMEUserPrivateKeyCreated
func (c *Controller) rectifyPhase2(logger dlog.Logger, acmeHosts []*ambassadorTypesV2.Host) []*ambassadorTypesV2.Host {
	var nextPhase []*ambassadorTypesV2.Host

	acmeHostsByPrivateKeySecret := make(map[string]map[string][]*ambassadorTypesV2.Host)
	for _, host := range acmeHosts {
		if _, nsSeen := acmeHostsByPrivateKeySecret[host.GetNamespace()]; !nsSeen {
			acmeHostsByPrivateKeySecret[host.GetNamespace()] = make(map[string][]*ambassadorTypesV2.Host)
		}
		acmeHostsByPrivateKeySecret[host.GetNamespace()][host.Spec.AcmeProvider.PrivateKeySecret.Name] = append(acmeHostsByPrivateKeySecret[host.GetNamespace()][host.Spec.AcmeProvider.PrivateKeySecret.Name], host)
	}

	// Act on 'acmeHostsByPrivateKeySecret'
	// Populate 'acmeHostsbyTLSSecret'
	logger.Debugln("rectify: Phase 1→2 (ACME account pre-registration): DefaultsFilled→ACMEUserPrivateKeyCreated")
	for namespace := range acmeHostsByPrivateKeySecret {
		logger := logger.WithField("namespace", namespace)
		for privateKeySecretName, hosts := range acmeHostsByPrivateKeySecret[namespace] {
			logger.Debugf("rectify: processing hosts that share private key Secret=%q: %v",
				privateKeySecretName,
				func() []string {
					hostResourceNames := make([]string, 0, len(hosts))
					for _, host := range hosts {
						hostResourceNames = append(hostResourceNames, host.GetName())
					}
					sort.Strings(hostResourceNames)
					return hostResourceNames
				}())
			// part 1: write the 'Secret'
			secret := c.getSecret(namespace, privateKeySecretName)
			if secret == nil {
				logger.Debugln("rectify: Secret: creating user private key")
				c.recordHostsEvent(hosts, "creating private key Secret")
				secret, err := generateUserPrivateKeySecret(namespace, privateKeySecretName)
				if err != nil {
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
						err)
				}
				for _, host := range hosts {
					secretAddOwner(secret, host)
				}
				if err := storeSecret(c.secretsGetter, secret); err != nil {
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
						err)
					continue
				}
				c.recordHostsEvent(hosts, "waiting for private key Secret creation to be reflected in snapshot")
				continue
			} else {
				secret = secret.DeepCopy()
				secretIsDirty := false
				for _, host := range hosts {
					if !secretIsOwnedBy(secret, host) {
						secretAddOwner(secret, host)
						secretIsDirty = true
					}
				}
				if secretIsDirty {
					logger.Debugln("rectify: Secret: updating ownership of user private key")
					c.recordHostsEvent(hosts, "modifying private key Secret")
					if err := storeSecret(c.secretsGetter, secret); err != nil {
						c.recordHostsError(logger, hosts,
							ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
							err)
						continue
					}
					c.recordHostsEvent(hosts, "waiting for private key Secret modification to be reflected in snapshot")
					continue
				}
			}
			// part 2: write the 'HostStatus'es
			hostsDirty := false
			for _, host := range hosts {
				if host.Status.State == ambassadorTypesV2.HostState_Pending && host.Status.PhaseCompleted < ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated {
					logger.Debugln("rectify: Secret: updating HostStatuses")
					c.recordHostPending(logger, host,
						ambassadorTypesV2.HostPhase_ACMEUserPrivateKeyCreated,
						ambassadorTypesV2.HostPhase_ACMEUserRegistered,
						"waiting for Host status change to be reflected in snapshot")
					hostsDirty = true
				}
			}
			if hostsDirty {
				continue
			}
			// part 4: continue to next phase
			logger.Debugln("rectify: Secret: accepting Hosts for next phase")
			nextPhase = append(nextPhase, hosts...)
		}
	}

	return nextPhase
}

// Phase 2→3 (ACME account registration): ACMEUserPrivateKeyCreated→ACMEUserRegistered
func (c *Controller) rectifyPhase3(logger dlog.Logger, acmeHosts []*ambassadorTypesV2.Host) (
	acmeHostsByTLSSecret map[string]map[string][]*ambassadorTypesV2.Host,
	acmeProviderByTLSSecret map[string]map[string]*ambassadorTypesV2.ACMEProviderSpec,
) {
	acmeHostsByTLSSecret = make(map[string]map[string][]*ambassadorTypesV2.Host)
	for _, host := range acmeHosts {
		if _, nsSeen := acmeHostsByTLSSecret[host.GetNamespace()]; !nsSeen {
			acmeHostsByTLSSecret[host.GetNamespace()] = make(map[string][]*ambassadorTypesV2.Host)
		}
		acmeHostsByTLSSecret[host.GetNamespace()][host.Spec.TlsSecret.Name] = append(acmeHostsByTLSSecret[host.GetNamespace()][host.Spec.TlsSecret.Name], host)
	}

	// Act on 'acmeHostsbyTLSSecret'
	// Populate 'acmeProviderByTLSSecret'
	logger.Debugln("rectify: Phase 2→3 (ACME account registration): ACMEUserPrivateKeyCreated→ACMEUserRegistered")
	acmeProviderByTLSSecret = make(map[string]map[string]*ambassadorTypesV2.ACMEProviderSpec)
	for namespace := range acmeHostsByTLSSecret {
		logger := logger.WithField("namespace", namespace)
		acmeProviderByTLSSecret[namespace] = make(map[string]*ambassadorTypesV2.ACMEProviderSpec)
		for tlsSecretName := range acmeHostsByTLSSecret[namespace] {
			logger := logger.WithField("secret", tlsSecretName)
			logger.Debugf("rectify: processing Hosts that share Secret=%q: %v",
				tlsSecretName,
				func() []string {
					hostResourceNames := make([]string, 0, len(acmeHostsByTLSSecret[namespace][tlsSecretName]))
					for _, host := range acmeHostsByTLSSecret[namespace][tlsSecretName] {
						hostResourceNames = append(hostResourceNames, host.GetName())
					}
					sort.Strings(hostResourceNames)
					return hostResourceNames
				}())
			hostsByProvider := make(map[providerKey][]*ambassadorTypesV2.Host)
			for _, host := range acmeHostsByTLSSecret[namespace][tlsSecretName] {
				hashKey := providerKey{
					Authority:            host.Spec.AcmeProvider.Authority,
					Email:                host.Spec.AcmeProvider.Email,
					PrivateKeySecretName: host.Spec.AcmeProvider.PrivateKeySecret.Name,
				}
				hostsByProvider[hashKey] = append(hostsByProvider[hashKey], host)
			}
			dirty := false
			for _, hosts := range hostsByProvider {
				logger.Debugf("rectify: Secret: processing Hosts that share provider (%q, %q): %v",
					hosts[0].Spec.AcmeProvider.Authority,
					hosts[0].Spec.AcmeProvider.Email,
					func() []string {
						hostResourceNames := make([]string, 0, len(hosts))
						for _, host := range hosts {
							hostResourceNames = append(hostResourceNames, host.GetName())
						}
						sort.Strings(hostResourceNames)
						return hostResourceNames
					}())

				registration := ""
				for _, host := range hosts {
					if host.Spec.AcmeProvider.Registration != "" {
						if registration != "" && registration != host.Spec.AcmeProvider.Registration {
							c.eventLogger.Namespace(host.GetNamespace()).Eventf(unstructureHost(host), k8sTypesCoreV1.EventTypeWarning, "Warning",
								"Host has disagreeing ACME registration from other Hosts with the same ACME credentials: %q",
								host.Spec.AcmeProvider.Registration)
							logger.Warningf("rectify: Secret: provider: host=%q has disagreeing ACME registration: %q",
								host.GetName(), host.Spec.AcmeProvider.Registration)
						}
						registration = host.Spec.AcmeProvider.Registration
					}
				}
				if registration != "" {
					logger.Debugln("rectify: Secret: provider: found existing registration")
				} else {
					logger.Debugln("rectify: Secret: provider: registering ACME user...")
					c.recordHostsEvent(hosts, "registering ACME account")
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
					c.recordHostsEvent(hosts, "ACME account registered")
				}
				for _, host := range hosts {
					logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
					logger.Debugf("rectify: Secret: provider: updating registration on Host=%q", host.GetName())
					if host.Spec.AcmeProvider.Registration != registration {
						host.Spec.AcmeProvider.Registration = registration
						c.recordHostPending(logger, host,
							ambassadorTypesV2.HostPhase_ACMEUserRegistered,
							ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
							"waiting for Host ACME account registration change to be reflected in snapshot")
						dirty = true
					}
				}
			}
			if dirty {
				logger.Debugln("rectify: Secret: 1 or more hosts changed, ignoring Secret until next snapshot")
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
				for _, host := range acmeHostsByTLSSecret[namespace][tlsSecretName] {
					c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeWarning, "Warning", "Host specified an 'acmeProvider' that differs from other Hosts with the same 'tlsSecret'")
				}
				logger.Warningln("there were multiple ACME providers specified for this secret")
			}
			logger.Debugln("rectify: Secret: accepting Hosts for next phase")
			acmeProviderByTLSSecret[namespace][tlsSecretName] = hostsByProvider[providerKeys[0]][0].Spec.AcmeProvider
		}
	}

	return acmeHostsByTLSSecret, acmeProviderByTLSSecret
}

// Phase 3→4→0 (ACME certificate request): ACMEUserRegistered→ACMECertificateChallenge→NA(state=Ready)
func (c *Controller) rectifyPhase4(logger dlog.Logger,
	acmeHostsByTLSSecret map[string]map[string][]*ambassadorTypesV2.Host,
	acmeProviderByTLSSecret map[string]map[string]*ambassadorTypesV2.ACMEProviderSpec,
) {
	// Act on 'acmeProviderByTLSSecret' and 'acmeHostsByTLSSecret'
	logger.Debugln("rectify: Phase 3→4→0 (ACME certificate request): ACMEUserRegistered→ACMECertificateChallenge→NA(state=Ready)")
	for namespace := range acmeProviderByTLSSecret {
		logger := logger.WithField("namespace", namespace)
		for tlsSecretName := range acmeProviderByTLSSecret[namespace] {
			hosts := acmeHostsByTLSSecret[namespace][tlsSecretName]
			logger := logger.WithField("secret", tlsSecretName)
			hostnames := make([]string, 0, len(hosts))
			for _, host := range hosts {
				hostnames = append(hostnames, host.Spec.Hostname)
			}
			sort.Strings(hostnames)
			logger.Debugf("rectify: processing Secret=%q (hostnames=%v)", tlsSecretName, hostnames)

			needsRenewReason := ""
			secretIsDirty := false

			secret := c.getSecret(namespace, tlsSecretName)
			if secret == nil {
				// "renew" certs that we don't even have an old version of
				needsRenewReason = "tlsSecret does not exist"
				secret = &k8sTypesCoreV1.Secret{
					ObjectMeta: k8sTypesMetaV1.ObjectMeta{
						Name:      tlsSecretName,
						Namespace: namespace,
					},
					Type: k8sTypesCoreV1.SecretTypeTLS,
				}
				secretIsDirty = true
			} else {
				secret = secret.DeepCopy()
				if cert, err := parseTLSSecret(secret); err != nil {
					// "renew" invalid certs
					needsRenewReason = fmt.Sprintf("tlsSecret doesn't appear to contain a valid TLS certificate: %v", err)
				} else if !stringSliceEqual(subjects(cert), hostnames) {
					// or if the list of hostnames we want on it changed
					needsRenewReason = fmt.Sprintf("list of desired host names changed: desired=%q certificate=%q", hostnames, subjects(cert))
				} else {
					// renew certs if they're >2/3 of the way through their lifecycle
					now := time.Now()
					age := now.Sub(cert.NotBefore)
					lifespan := cert.NotAfter.Sub(cert.NotBefore)
					if age > 2*lifespan/3 {
						needsRenewReason = fmt.Sprintf("certificate is more than 2/3 of the way to expiration: %v is %d%% of the way from %v to %v",
							now,
							100*int64(age)/int64(lifespan),
							cert.NotBefore,
							cert.NotAfter)
					}
				}
			}

			logger.Debugf("rectify: Secret: needsRenewReason=%v", needsRenewReason)
			if needsRenewReason != "" {
				c.recordHostsEvent(hosts, fmt.Sprintf("tlsSecret %q.%q (hostnames=%q): needs updated: %v",
					tlsSecretName, namespace, hostnames,
					needsRenewReason))
				acmeProvider := acmeProviderByTLSSecret[namespace][tlsSecretName]
				var user acmeUser
				var err error
				user.Email = acmeProvider.Email
				user.PrivateKey, err = parseUserPrivateKey(c.getSecret(namespace, acmeProvider.PrivateKeySecret.Name))
				if err != nil {
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
						err)
					continue
				}
				var reg registration.Resource
				if err = json.Unmarshal([]byte(acmeProvider.Registration), &reg); err != nil {
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
						err)
					continue
				}
				user.Registration = &reg

				logger.Debugln("rectify: Secret: requesting certificate...")
				c.recordHostsEvent(hosts, fmt.Sprintf("performing ACME challenge for tlsSecret %q.%q (hostnames=%q)...",
					tlsSecretName, namespace, hostnames))
				certResource, err := obtainCertificate(
					c.httpClient,
					c.redisPool,
					acmeProvider.Authority,
					&user,
					hostnames)
				if err != nil {
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
						errors.Wrapf(err, "obtaining tlsSecret %q.%q (hostnames=%q)",
							tlsSecretName, namespace, hostnames))
					continue
				}
				secret.Data = map[string][]byte{
					"tls.key": certResource.PrivateKey,
					"tls.crt": certResource.Certificate,
				}
				secretIsDirty = true
			}
			for _, host := range hosts {
				if !secretIsOwnedBy(secret, host) {
					secretAddOwner(secret, host)
					secretIsDirty = true
				}
			}
			if secretIsDirty {
				logger.Debugln("rectify: Secret: updating Secret")
				c.recordHostsEvent(hosts, "updating TLS Secret")
				if err := storeSecret(c.secretsGetter, secret); err != nil {
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
						errors.Wrapf(err, "updating tlsSecret %q.%q (hostnames=%q)",
							tlsSecretName, namespace, hostnames))
					continue
				}
				c.recordHostsEvent(hosts, "waiting for TLS Secret update to be reflected in snapshot")
				continue
			}
			logger.Debugln("rectify: Secret: updating HostStatuses")
			for _, host := range acmeHostsByTLSSecret[namespace][tlsSecretName] {
				if host.Status.State != ambassadorTypesV2.HostState_Ready {
					logger := logger.WithField("host", host.GetName()+"."+host.GetNamespace())
					c.recordHostReady(logger, host, "Host with ACME-provisioned TLS certificate marked Ready")
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
			//host.Spec.TlsSecret.Name = NameEncode(host.Spec.AcmeProvider.Authority) + "--" + NameEncode(host.Spec.AcmeProvider.Email) + "--" + NameEncode(host.Spec.AcmeProvider.PrivateKeySecret.Name)
			host.Spec.TlsSecret.Name = NameEncode(host.Spec.Hostname)
		}
	}
	if host.Status == nil {
		host.Status = &ambassadorTypesV2.HostStatus{}
	}
	switch {
	case host.Spec.AcmeProvider.Authority != "none":
		host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_ACME
	case host.Spec.TlsSecret.Name == "":
		host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_Other
	default:
		host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_None
	}
}
