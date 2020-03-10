package acmeclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-acme/lego/v3/registration"
	"github.com/gogo/protobuf/proto"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mholt/certmagic"
	"github.com/pkg/errors"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sClientDynamic "k8s.io/client-go/dynamic"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/events"
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/leaderelection"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
)

type ref struct {
	Name      string
	Namespace string
}

type Controller struct {
	cfg      types.Config
	kubeinfo *k8s.KubeInfo

	redisPool  *pool.Pool
	httpClient *http.Client

	snapshotCh  <-chan watt.Snapshot
	eventLogger *events.EventLogger

	secretsGetter k8sClientCoreV1.SecretsGetter
	hostsGetter   k8sClientDynamic.NamespaceableResourceInterface

	hosts               []*ambassadorTypesV2.Host
	knownChangedHosts   map[ref]struct{}
	secrets             []*k8sTypesCoreV1.Secret
	knownChangedSecrets map[ref]struct{}
}

func NewController(
	cfg types.Config,
	kubeinfo *k8s.KubeInfo,
	redisPool *pool.Pool,
	httpClient *http.Client,
	snapshotCh <-chan watt.Snapshot,
	eventLogger *events.EventLogger,
	secretsGetter k8sClientCoreV1.SecretsGetter,
	dynamicClient k8sClientDynamic.Interface,
) *Controller {
	return &Controller{
		cfg:         cfg,
		kubeinfo:    kubeinfo,
		redisPool:   redisPool,
		httpClient:  httpClient,
		snapshotCh:  snapshotCh,
		eventLogger: eventLogger,

		secretsGetter: secretsGetter,
		hostsGetter:   dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "hosts"}),

		knownChangedHosts:   make(map[ref]struct{}),
		knownChangedSecrets: make(map[ref]struct{}),
	}
}

func (c *Controller) anyInconsistent() bool {
	return len(c.knownChangedHosts) > 0 || len(c.knownChangedSecrets) > 0
}

func (c *Controller) Worker(ctx context.Context) error {
	ctx, cancelElection := context.WithCancel(ctx)
	err := leaderelection.RunAsSingleton(ctx, c.cfg, c.kubeinfo, "acmeclient", 60*time.Second, func(ctx context.Context) {
		// ctx will be canceled when we are no longer the leader (or are shutting
		// down).

		logger := dlog.GetLogger(ctx)

		// What follows is a simple event-loop.  It allows us to write our rectify() and
		// processSnapshot() functions in a simple (well, simpler) single-threaded manner,
		// without having to worry about multiple goroutines or signalling or anything.
		//
		// "Ideally", there would be 3 types of events in this event loop:
		//  1. A new WATT snapshot
		//  2. A timed event; analogous to JavaScript's setTimeout()
		//  3. A periodic ticker, to avoid becoming wedged in case something goes sideways and
		//     we forget to call that setTimeout()-analogue.
		//
		// "Naturally", I'd set the (3) ticker to 24 hours.
		//
		// However, it's a gap in the implementation that we don't actually have the (2)
		// setTimeout()-analogue.  I wrote the original implementation with just (1) and (3),
		// and now adding (2) is a little tricky.  Or at least tedious.  The point is, I didn't
		// have time to add it with 1.0.0 GA around the corner.
		//
		// Why do we want a (2) setTimeout()-analogue, when I didn't need one for the initial
		// implementation?  The big reason is that we added errorBackoff, and we really want to
		// call rectify() again when an errorBackoff expires.
		//
		// So, what to do about that: As a stop-gap for not having a (2) setTimeout()-analogue:
		// Burn some CPU cycles, and crank the (3) ticker down from daily to minutely, so that
		// we always trigger within a minute of an errorBackoff elapsing, at the cost of a bunch
		// of no-op calls to c.rectify().
		//
		// A no-op c.rectify() should be cheap enough (entirely in-CPU) that until I see a
		// benchmark saying otherwise, doing this the "right way" and adding that (2)
		// setTimeout()-analogue is pretty low priority.
		ticker := time.NewTicker(time.Minute) // 24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// we are no longer the leader--bail out
				return
			case <-ticker.C:
				// It seems like it should be a good idea to trigger another rectify
				// here -- but wait!  If we have any Hosts or Secrets that are known to
				// have changes that aren't yet observed in the WATT snapshot, then it's
				// not safe to rectify, so don't do anything in that case.
				if !c.anyInconsistent() {
					// It's safe to rectify! Off we go.
					logger.Infoln("triggering rectify from timer...")
					c.rectify(logger)
				} else {
					logger.Infoln("skipping rectify from timer due to inconsistent snapshot...")
				}
			case snapshot, ok := <-c.snapshotCh:
				if !ok {
					// there's no more work to do; not only do we want the current
					// OnStartedLeading callback to return, we walso want the
					// leaderElector.Run() to return; so we cancel the election.
					cancelElection()
					return
				}
				logger.Debugln("processing snapshot change...")
				if c.processSnapshot(snapshot, logger) {
					c.rectify(logger)
				}
			}
		}
	})
	if err != nil {
		// make this non-fatal
		dlog.GetLogger(ctx).Errorln("failed to participate in acme leader election, Ambassador Edge Stack ACME client is disabled:", err)
	}
	return nil
}

func getHostResourceVersion(hosts []*ambassadorTypesV2.Host, ref ref) string {
	for _, host := range hosts {
		if host.GetNamespace() == ref.Namespace && host.GetName() == ref.Name {
			return host.GetResourceVersion()
		}
	}
	return ""
}

func getSecretResourceVersion(secrets []*k8sTypesCoreV1.Secret, ref ref) string {
	for _, secret := range secrets {
		if secret.GetNamespace() == ref.Namespace && secret.GetName() == ref.Name {
			return secret.GetResourceVersion()
		}
	}
	return ""
}

func (c *Controller) processSnapshot(snapshot watt.Snapshot, logger dlog.Logger) (changed bool) {
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
		// If there are any Hosts or Secrets that we know to have changed, but haven't yet observed that change
		// in this WATT snapshot, then discard this snapshot and wait for a sufficiently up-to-date one.
		for hostRef := range c.knownChangedHosts {
			if getHostResourceVersion(hosts, hostRef) == getHostResourceVersion(c.hosts, hostRef) {
				logger.Debugln("snapshot does not include required change to Host", hostRef)
				return false
			}
			logger.Debugln("observed required change to Host", hostRef)
			delete(c.knownChangedHosts, hostRef)
		}
		for secretRef := range c.knownChangedSecrets {
			if getSecretResourceVersion(secrets, secretRef) == getSecretResourceVersion(c.secrets, secretRef) {
				logger.Debugln("snapshot does not include required change to Secret", secretRef)
				return false
			}
			logger.Debugln("observed required change to Secret", secretRef)
			delete(c.knownChangedSecrets, secretRef)
		}

		// OK, the snapshot is sufficiently up-to-date, and contains new info.  Update our view of the world.
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
	var err error
	uHost := unstructureHost(host)

	uHost, err = c.hostsGetter.Namespace(host.GetNamespace()).Update(uHost, k8sTypesMetaV1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "update %q.%q", host.GetName(), host.GetNamespace())
	}
	if uHost.GetResourceVersion() != host.GetResourceVersion() {
		c.knownChangedHosts[ref{Name: host.GetName(), Namespace: host.GetNamespace()}] = struct{}{}
	}
	uHost.Object["status"] = host.Status

	uHost, err = c.hostsGetter.Namespace(host.GetNamespace()).UpdateStatus(uHost, k8sTypesMetaV1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "updateStatus %q.%q", host.GetName(), host.GetNamespace())
	}
	if uHost.GetResourceVersion() != host.GetResourceVersion() {
		c.knownChangedHosts[ref{Name: host.GetName(), Namespace: host.GetNamespace()}] = struct{}{}
	}

	return err
}

// Hear ye, hear ye: Immediately after any of the `.recordHost*` methods except for `recordHostsEvent`, you MUST avoid
// further processing of that Host/those Hosts until you have a new WATT snapshot reflecting the changes.  Because of
// the way that each of the `rectifyPhaseXXX` methods are structured, that pretty much just means calling `continue`
// afterward.
//
// So, when mechanically reviewing the code, you should either (1) look for calling `continue` immediately after the
// call to the `recordHostXXX` method, like:
//
// 	c.recordHostPending(logger, host,
// 		ambassadorTypesV2.HostPhase_DefaultsFilled,
// 		nextPhase, "waiting for Host DefaultsFilled change to be reflected in snapshot")
// 	continue
// 	// more code that would process the Host would go here
//
// or (2) you should look for calling `continue` immediately after a loop that called the `recordHostXXX` method on a
// set of Hosts, like:
//
// 	hostsDirty := false
// 	for _, host := range hosts {
// 		if shouldUpdateHost(host) {
// 			c.recordHostPending(logger, host, ...)
// 			hostsDirty = true
// 		}
// 	}
// 	if hostsDirty {
// 		continue
// 	}
// 	// more code that would process the Hosts would go here
//
// And that rule of "make sure you call `continue` afterward" should save you the mental overhead of having to think too
// much about it.

// recordHostPending records a Host as state=Pending with the given details (potentially moving
// it out of state=Error or state=Ready).
//
// After calling this method, you MUST not process the Host further until you get a new
// snapshot reflecting the change.
func (c *Controller) recordHostPending(logger dlog.Logger, host *ambassadorTypesV2.Host, phaseCompleted, phasePending ambassadorTypesV2.HostPhase, reasonPending string) {
	logger.Debugf("updating pending host %d→%d", host.Status.PhaseCompleted, phaseCompleted)
	if phaseCompleted <= host.Status.PhaseCompleted {
		logger.Debugf("^^ THIS IS A BUG ^^: %d→%d is not a progression", host.Status.PhaseCompleted, phaseCompleted)
	}
	host.Status.State = ambassadorTypesV2.HostState_Pending
	host.Status.PhaseCompleted = phaseCompleted
	host.Status.PhasePending = phasePending
	host.Status.ErrorReason = ""
	host.Status.ErrorTimestamp = nil
	host.Status.ErrorBackoff = nil
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
		return
	}
	c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeNormal, "Pending", reasonPending)
}

// recordHostReady records a Host as state=Ready (potentially moving it out of state=Error or
// state=Pending).
//
// After calling this method, you MUST not process the Host further until you get a new
// snapshot reflecting the change.
func (c *Controller) recordHostReady(logger dlog.Logger, host *ambassadorTypesV2.Host, readyReason string) {
	logger.Debugln("updating ready host")
	host.Status.State = ambassadorTypesV2.HostState_Ready
	host.Status.PhaseCompleted = ambassadorTypesV2.HostPhase_NA
	host.Status.PhasePending = ambassadorTypesV2.HostPhase_NA
	host.Status.ErrorReason = ""
	host.Status.ErrorTimestamp = nil
	host.Status.ErrorBackoff = nil
	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
		return
	}
	c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeNormal, "Ready", readyReason)
}

// recordHostError records a Host as state=Error with the given details (potentially moving it
// out of state=Ready or state=Pending).
//
// After calling this method, you MUST not process the Host further until you get a new
// snapshot reflecting the change.
func (c *Controller) recordHostError(logger dlog.Logger, host *ambassadorTypesV2.Host, phase ambassadorTypesV2.HostPhase, err error) {
	logger.Debugln("updating errored host:", err)
	host.Status.State = ambassadorTypesV2.HostState_Error
	host.Status.PhasePending = phase

	host.Status.ErrorReason = err.Error()

	now := time.Now()
	host.Status.ErrorTimestamp = &now

	var prevBackoff time.Duration
	if host.Status.ErrorBackoff != nil {
		prevBackoff = *host.Status.ErrorBackoff
	}

	// This heuristic tries to detect whether the error is that the ACME
	// provider got NXDOMAIN for the provided hostname. It specifically handles
	// the error message returned by Let's Encrypt in Feb 2020, but it may cover
	// others as well. The goal is to try ACME again much sooner for this case,
	// as we expect NXDOMAIN to be resolved quickly. We ran into this when
	// giving users *.edgestack.me domain names with `edgectl install`.
	isAcmeNxDomain := strings.Contains(host.Status.ErrorReason, "NXDOMAIN") || strings.Contains(host.Status.ErrorReason, "urn:ietf:params:acme:error:dns")

	nextBackoff := getNextBackoff(prevBackoff, isAcmeNxDomain)
	host.Status.ErrorBackoff = &nextBackoff

	if err := c.updateHost(host); err != nil {
		logger.Errorln(err)
		return
	}
	c.eventLogger.Namespace(host.GetNamespace()).Event(unstructureHost(host), k8sTypesCoreV1.EventTypeWarning, "Error", err.Error())
}

// recordHostsError is a convenience wrapper around recordHostError, and calls recordHostError for each of the listed Hosts.
//
// After calling this method, you MUST not process any of the listed Hosts further until you
// get a new snapshot reflecting the change.
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

		host.Spec = getEffectiveSpec(host)
		if host.Status == nil {
			host.Status = &ambassadorTypesV2.HostStatus{}
		}
		switch {
		case host.Spec.AcmeProvider.Authority != "none":
			// TLS using via AES ACME integration
			host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_ACME
		case host.Spec.TlsSecret.Name != "":
			// TLS configured via some other mechanism
			host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_Other
		default:
			// No TLS
			host.Status.TlsCertificateSource = ambassadorTypesV2.HostTLSCertificateSource_None
		}
		if !proto.Equal(host.Spec, _host.Spec) || !proto.Equal(host.Status, _host.Status) {
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

		if host.Status.State == ambassadorTypesV2.HostState_Error &&
			host.Status.ErrorTimestamp != nil && host.Status.ErrorBackoff != nil &&
			time.Now().Before(host.Status.ErrorTimestamp.Add(*host.Status.ErrorBackoff)) {
			logger.Debugln("rectify: Host: in error backoff; skipping")
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
			if !certmagic.HostQualifies(host.Spec.Hostname) {
				c.recordHostError(logger, host,
					ambassadorTypesV2.HostPhase_NA,
					errors.Errorf("hostname=%q does not qualify for ACME management", host.Spec.Hostname))
			} else {
				logger.Debugln("rectify: Host: accepting Host for next phase")
				nextPhase = append(nextPhase, host)
			}
		default:
			// Even if the user filled in an invalid TlsCertificateSource with kubectl or something,
			// FillDefaults should have corrected it by the time we make it to this part of the code.
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
					continue
				}
				for _, host := range hosts {
					secretAddOwner(secret, host)
				}
				if err := c.storeSecret(c.secretsGetter, secret); err != nil {
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
					if err := c.storeSecret(c.secretsGetter, secret); err != nil {
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
				now := time.Now()
				if cert, err := parseTLSSecret(secret); err != nil {
					// "renew" invalid certs
					needsRenewReason = fmt.Sprintf("tlsSecret doesn't appear to contain a valid TLS certificate: %v", err)
				} else if !stringSliceEqual(subjects(cert), hostnames) {
					// or if the list of hostnames we want on it changed
					needsRenewReason = fmt.Sprintf("list of desired host names changed: desired=%q certificate=%q", hostnames, subjects(cert))
				} else if age, lifespan := now.Sub(cert.NotBefore), cert.NotAfter.Sub(cert.NotBefore); age > 2*lifespan/3 {
					// renew certs if they're >2/3 of the way through their lifecycle
					needsRenewReason = fmt.Sprintf("certificate is more than 2/3 of the way to expiration: %v is %d%% of the way from %v to %v",
						now,
						100*int64(age)/int64(lifespan),
						cert.NotBefore,
						cert.NotAfter)
				}
			}

			logger.Debugf("rectify: Secret: needsRenewReason=%v", needsRenewReason)
			if needsRenewReason != "" {
				c.recordHostsEvent(hosts, fmt.Sprintf("tlsSecret %q.%q (hostnames=%q): needs updated: %v",
					tlsSecretName, namespace, hostnames,
					needsRenewReason))
				if c.redisPool == nil {
					// We need Redis to do the ACME challenge.  Why do this check so late, at the
					// leaves?
					//
					// In runner/main.go, we could avoid launching the ACME client at all if we
					// don't have Redis, but then we wouldn't get defaults fill or status fill for
					// non-ACME Hosts.
					//
					// We could just bail after rectifyPhase1 if we don't have Redis, but then the
					// user would have no indication why ACME wasn't working.
					//
					// We could instead record an error for ACME Hosts after rectifyPhase1, but then
					// we might interfere with valid Hosts that would have still been good for a
					// while, during a botched upgrade or something.
					//
					// So instead, we go through as much of the process as we can, until we actually
					// need Redis, and then record an error on the Hosts that would need Redis.
					c.recordHostsError(logger, hosts,
						ambassadorTypesV2.HostPhase_ACMECertificateChallenge,
						errors.New("redis is not configured"))
					continue
				}
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
				if err := c.storeSecret(c.secretsGetter, secret); err != nil {
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

func getNextBackoff(prevBackoff time.Duration, isAcmeNxdomain bool) time.Duration {
	// The letsencrypt ratelimit is 5 failures per account, per
	// hostname, per hour.  So we could be pretty safe with, say,
	// a 15m backoff.  But let's do an exponential backoff anyway,
	// starting at 10m, and maxing out at 24h.
	if prevBackoff == 0 {
		if isAcmeNxdomain {
			// If the ACME server responds that it got an
			// NXDOMAIN, then we want to start off with a
			// particularly small backoff, because it will
			// likely be resolved quickly...
			return 1 * time.Minute
		}
		return 5 * time.Minute
	}
	var ret time.Duration
	if isAcmeNxdomain {
		// ...but then we'll grow the backoff slightly more
		// aggressively, to avoid hitting the rate-limit.
		ret = prevBackoff * 3
	} else {
		ret = prevBackoff * 2
	}
	if ret > 24*time.Hour {
		return 24 * time.Hour
	}
	return ret
}

func getEffectiveSpec(host *ambassadorTypesV2.Host) *ambassadorTypesV2.HostSpec {
	hostSpec := deepCopyHostSpec(host.Spec)

	// Ensure that all nested structures exist, so we don't need to worry about nil pointers
	if hostSpec == nil {
		hostSpec = &ambassadorTypesV2.HostSpec{}
	}
	if hostSpec.Selector == nil {
		hostSpec.Selector = &k8sTypesMetaV1.LabelSelector{}
	}
	if hostSpec.AcmeProvider == nil {
		hostSpec.AcmeProvider = &ambassadorTypesV2.ACMEProviderSpec{}
	}
	if hostSpec.TlsSecret == nil {
		hostSpec.TlsSecret = &k8sTypesCoreV1.LocalObjectReference{}
	}

	// Now actually fill the values
	if hostSpec.AmbassadorId == nil { // XXX: should this be `len(hostSpec.AmbassadorId) == 0`?
		hostSpec.AmbassadorId = []string{"default"}
	}
	if hostSpec.Hostname == "" {
		hostSpec.Hostname = host.GetName()
	}
	if len(hostSpec.Selector.MatchLabels)+len(hostSpec.Selector.MatchExpressions) == 0 {
		hostSpec.Selector.MatchLabels = map[string]string{
			"hostname": hostSpec.Hostname,
		}
	}
	if hostSpec.AcmeProvider.Authority == "" {
		hostSpec.AcmeProvider.Authority = "https://acme-v02.api.letsencrypt.org/directory"
	}
	if hostSpec.AcmeProvider.Authority != "none" {
		if hostSpec.AcmeProvider.PrivateKeySecret == nil {
			hostSpec.AcmeProvider.PrivateKeySecret = &k8sTypesCoreV1.LocalObjectReference{}
		}
		if hostSpec.AcmeProvider.PrivateKeySecret.Name == "" {
			if hostSpec.AcmeProvider.Email == "" {
				hostSpec.AcmeProvider.PrivateKeySecret.Name = NameEncode(hostSpec.AcmeProvider.Authority)
			} else {
				hostSpec.AcmeProvider.PrivateKeySecret.Name = NameEncode(hostSpec.AcmeProvider.Authority) + "--" + NameEncode(hostSpec.AcmeProvider.Email)
			}
		}
		if hostSpec.TlsSecret.Name == "" {
			// hostSpec.TlsSecret.Name = NameEncode(hostSpec.AcmeProvider.Authority) + "--" + NameEncode(hostSpec.AcmeProvider.Email) + "--" + NameEncode(hostSpec.AcmeProvider.PrivateKeySecret.Name)
			hostSpec.TlsSecret.Name = NameEncode(hostSpec.Hostname)
		}
	}

	return hostSpec
}
