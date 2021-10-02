package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// The IstioCertSource and IstioCertWatcher interfaces exist to allow dependency
// injection while testing the watcher. What you see here are the production
// implementations:
//
// istioCertSource implements IstioCertSource: its Watch() method returns an
// istioCertWatcher, which implements IstioCertWatcher in turn.
type istioCertSource struct {
}

type istioCertWatcher struct {
	updateChannel chan IstioCertUpdate
}

func newIstioCertSource() IstioCertSource {
	return &istioCertSource{}
}

// Watch sets up to watch for an Istio cert on the filesystem, if need be. This
// is the production implementation, which returns an istioCertWatcher to implement
// the IstioCertWatcher interface.
func (src *istioCertSource) Watch(ctx context.Context) (IstioCertWatcher, error) {
	// We can watch the filesystem for Istio mTLS certificates. Here, we fire
	// up the stuff we need to do that -- specifically, we need an FSWatcher
	// to watch the filesystem, an IstioCert to manage the cert, and an update
	// channel to hear about new Istio stuff.
	//
	// The actual functionality here is currently keyed off the environment
	// variable AMBASSADOR_ISTIO_SECRET_DIR, but we set the update channel
	// either way to keep the select logic below simpler. If the environment
	// variable is unset, we never instantiate the FSWatcher or IstioCert,
	// so there will never be any updates on the update channel.
	istioCertUpdateChannel := make(chan IstioCertUpdate)

	// OK. Are we supposed to watch anything?
	secretDir := os.Getenv("AMBASSADOR_ISTIO_SECRET_DIR")

	if secretDir != "" {
		// Yup, get to it. First, fire up the IstioCert, and tell it to
		// post to our update channel from above.
		icert := NewIstioCert(secretDir, "istio-certs", GetAmbassadorNamespace(), istioCertUpdateChannel)

		// Next up, fire up the FSWatcher...
		fsw, err := NewFSWatcher(ctx)
		if err != nil {
			return nil, err
		}

		// ...then tell the FSWatcher to watch the Istio cert directory,
		// and give it a handler function that'll update the IstioCert
		// in turn.
		//
		// XXX This handler function is really just an impedance matcher.
		// Maybe IstioCert should just have a "HandleFSWEvent"...
		err = fsw.WatchDir(ctx, secretDir,
			func(ctx context.Context, event FSWEvent) {
				// Is this a deletion?
				deleted := (event.Op == FSWDelete)

				// OK. Feed this event into the IstioCert.
				icert.HandleEvent(ctx, event.Path, deleted)
			},
		)
		if err != nil {
			dlog.Errorf(ctx, "FileSystemWatcher.WatchDir(ctx, %q, fn) => %v",
				secretDir, err)
		}
	}

	return &istioCertWatcher{
		updateChannel: istioCertUpdateChannel,
	}, nil
}

// Changed returns the channel where Istio certificates will appear.
func (istio *istioCertWatcher) Changed() chan IstioCertUpdate {
	return istio.updateChannel
}

// istioCertWatchManager is the interface between all the Istio-cert-watching stuff
// and the watcher (in watcher.go).
type istioCertWatchManager struct {
	// XXX Temporary hack: we currently store the secrets found by the Istio-cert
	// watcher in the K8s snapshot, but this gives the Istio-cert watcher an easy
	// way to note that it saw changes. This is important because if any of the
	// watchers see changes, we can't short-circuit the reconfiguration.
	watcher        IstioCertWatcher
	changesPresent bool
}

// Changed returns a channel to listen on for change notifications dealing with
// Istio cert stuff.
func (imgr *istioCertWatchManager) Changed() chan IstioCertUpdate {
	return imgr.watcher.Changed()
}

// Update actually does the work of updating our internal state with changes. The
// istioCertWatchManager isn't allowed to short-circuit early: it's assumed that
// any update is relevant.
func (imgr *istioCertWatchManager) Update(ctx context.Context, icertUpdate IstioCertUpdate, k8sSnapshot *snapshot.KubernetesSnapshot) {
	dlog.Debugf(ctx, "WATCHER: ICert fired")

	// We've seen a change in the Istio cert info on the filesystem. This is
	// kind of a hack, but let's just go ahead and say that if we see an event
	// here, it's a real change -- presumably we won't be told to watch Istio
	// certs if they aren't important.
	//
	// XXX Obviously this is a crock and we should actually track whether the
	// secret is in use.
	imgr.changesPresent = true

	// Make a SecretRef for this new secret...
	ref := snapshotTypes.SecretRef{Name: icertUpdate.Name, Namespace: icertUpdate.Namespace}

	// ...and delete or save, as appropriate.
	if icertUpdate.Op == "delete" {
		dlog.Infof(ctx, "IstioCert: certificate %s.%s deleted", icertUpdate.Name, icertUpdate.Namespace)
		delete(k8sSnapshot.FSSecrets, ref)
	} else {
		dlog.Infof(ctx, "IstioCert: certificate %s.%s updated", icertUpdate.Name, icertUpdate.Namespace)
		k8sSnapshot.FSSecrets[ref] = icertUpdate.Secret
	}
	// Once done here, k8sSnapshot.ReconcileSecrets will handle the rest.
}

// StartLoop sets up the istioCertWatchManager for the start of the watcher loop.
func (imgr *istioCertWatchManager) StartLoop(ctx context.Context) {
	// Start every loop by assuming that no changes are present.
	imgr.changesPresent = false
}

// UpdatesPresent returns whether or not any significant updates have actually
// happened.
func (imgr *istioCertWatchManager) UpdatesPresent() bool {
	return imgr.changesPresent
}

// newIstioCertWatchManager returns... a new istioCertWatchManager.
func newIstioCertWatchManager(ctx context.Context, watcher IstioCertWatcher) *istioCertWatchManager {
	istio := istioCertWatchManager{
		watcher:        watcher,
		changesPresent: false,
	}

	return &istio
}

// Istio TLS certificates are annoying. They come in three parts (only two of
// which are needed), they're updated non-atomically, and we need to make sure we
// don't try to reconfigure when the parts are out of sync. Therefore, we keep
// track of the last-update time of both parts, and only update once both have
// been updated at the "same" time.

type pemReader func(ctx context.Context, dir string, name string) ([]byte, bool)
type timeFetcher func() time.Time

// IstioCert holds all the state we need to manage an Istio certificate.
type IstioCert struct {
	dir        string
	name       string // Name we'll use when generating our secret
	namespace  string // Namespace in which our secret will appear to be
	timestamps map[string]time.Time

	// How shall we read PEM files?
	readPEM pemReader

	// How shall we fetch the current time?
	fetchTime timeFetcher

	// Where shall we send updates when things happen?
	updates chan IstioCertUpdate
}

// IstioCertUpdate gets sent over the IstioCert's Updates channel
// whenever the cert changes
//
// XXX This will morph into a more general "internally-generated resource"
// thing later.
type IstioCertUpdate struct {
	Op        string        // "update" or "delete"
	Name      string        // secret name
	Namespace string        // secret namespace
	Secret    *kates.Secret // IstioCert secret
}

// NewIstioCert instantiates an IstioCert to manage a certificate that Istio
// will write into directory "dir", should have the given "name" and appear
// to live in K8s namespace "namespace", and will have updates posted to
// "updateChannel" whenever the cert changes.
//
// What's with this namespace business? Well, Ambassador may be running in
// single-namespace mode, so causing our cert to appear to be in the same
// namespace as Ambassador will just be less confusing for everyone.
//
// XXX Nomenclature is a little odd here. Istio is writing a _certificate_,
// but we're supplying it to the rest of Ambassador as a thing that looks
// like a Kubernetes TLS _Secret_ -- so we call this class an IstioCert,
// but the thing it's posting to the updateChannel includes a kates.Secret.
// Names are hard.
func NewIstioCert(dir string, name string, namespace string, updateChannel chan IstioCertUpdate) *IstioCert {
	icert := &IstioCert{
		dir:       dir,
		name:      name,
		namespace: namespace,
		fetchTime: time.Now, // default to using time.Now for time
		updates:   updateChannel,
	}

	// Default to using our default PEM reader...
	icert.readPEM = icert.defaultReadPEM

	// ...initialize the timestamp map...
	icert.timestamps = make(map[string]time.Time)

	return icert
}

// String returns a string representation of this IstioCert.
func (icert *IstioCert) String() string {
	// Our dir may be nothing, if we're just a dummy IstioCert
	// that's being used to make other logic easier. If that's the
	// case, be a little more verbose here.

	if icert.dir == "" {
		return "IstioCert (noop)"
	}

	return fmt.Sprintf("IstioCert %s", icert.dir)
}

// SetFetchTime will change the function we use to get the current time.
func (icert *IstioCert) SetFetchTime(fetchTime timeFetcher) {
	icert.fetchTime = fetchTime
}

// SetReadPEM will change the function we use to read PEM files.
func (icert *IstioCert) SetReadPEM(readPEM pemReader) {
	icert.readPEM = readPEM
}

// defaultReadPEM is the same as ioutil.ReadFile, really, but it logs for us
// if anything goes wrong.
func (icert *IstioCert) defaultReadPEM(ctx context.Context, dir string, name string) ([]byte, bool) {
	pemPath := path.Join(dir, name)

	pem, err := ioutil.ReadFile(pemPath)

	if err != nil {
		dlog.Errorf(ctx, "%s: couldn't read %s: %s", icert, pemPath, err)
		return nil, false
	}

	return pem, true
}

// getTimeFor tries to look up the timestamp we have stored for a given key,
// but it logs if it's missing (for debugging).
func (icert *IstioCert) getTimeFor(ctx context.Context, name string) (time.Time, bool) {
	then, exists := icert.timestamps[name]

	if !exists {
		dlog.Debugf(ctx, "%s: %s missing", icert, name)
		return time.Time{}, false
	}

	return then, true
}

// HandleEvent tells an IstioCert to update its internal state because a file
// in its directory has been written. If all the cert files have been updated
// closely enough in time, Update will decide that it's time to actually update
// the cert, and it'll send an IstioCertUpdate over the Updates channel.
func (icert *IstioCert) HandleEvent(ctx context.Context, name string, deleted bool) {
	// Istio writes three files into its cert directory:
	// - key.pem is the private key
	// - cert-chain.pem is the public keychain
	// - root-cert.pem is the CA root public key
	//
	// We ignore root-cert.pem, because cert-chain.pem contains it, and we
	// ignore any other name because Istio shouldn't be writing it there.
	//
	// Start by splitting the incoming name (which is really a path) into its
	// component parts, just 'cause it (mostly) makes life easier to refer
	// to things by the basename (the key) rather than the full path.
	dir := path.Dir(name)
	key := path.Base(name)

	dlog.Debugf(ctx, "%s: updating %s at %s", icert, key, name)

	if dir != icert.dir {
		// Somehow this is in the wrong directory? Toss it.
		dlog.Debugf(ctx, "%s: ignoring %s in dir %s", icert, name, dir)
		return
	}

	if (key != "key.pem") && (key != "cert-chain.pem") {
		// Someone is writing a file we don't need. Toss it.
		dlog.Debugf(ctx, "%s: ignoring %s", icert, name)
		return
	}

	// If this is a deletion...
	if deleted {
		// ...then drop the key from our timestamps map.
		delete(icert.timestamps, key)
	} else {
		// Not a deletion -- update the time for this key.
		icert.timestamps[key] = icert.fetchTime()
	}

	// Do we have both key.pem and cert-chain.pem? (It's OK to just return immediately
	// without logging, if not, because getTime logs for us.)
	kTime, kExists := icert.getTimeFor(ctx, "key.pem")
	cTime, cExists := icert.getTimeFor(ctx, "cert-chain.pem")

	bothPresent := (kExists && cExists)

	// OK. Is it time to do anything?
	if bothPresent && !deleted {
		// Maybe. Everything we need is present, but are the updates close enough
		// in time? Start by finding out which of key & cert-chain is newest, so
		// that we can find the delta between them.
		//
		// XXX Wouldn't it be nice if time.Duration had an AbsValue method?t
		var delta time.Duration

		if cTime.Before(kTime) {
			delta = kTime.Sub(cTime)
		} else {
			delta = cTime.Sub(kTime)
		}

		// OK, if the delta is more than five seconds (which is crazy long), we're done.
		//
		// Why five? Well, mostly just 'cause it's really easy to imagine these getting
		// written on different sides of a second boundary, really hard to imagine it
		// taking longer than five seconds, and really hard to imagine trying to rotate
		// certs every few seconds instead of every few minutes...

		if delta > 5*time.Second {
			dlog.Debugf(ctx, "%s: cert-chain/key delta %v, out of sync", icert, delta)
			return
		}

		// OK, times look good -- grab the JSON for this thing.
		secret, ok := icert.Secret(ctx)

		if !ok {
			// WTF.
			dlog.Debugf(ctx, "%s: cannot construct secret", icert)
		}

		// FINALLY!
		dlog.Debugf(ctx, "%s: noting update!", icert)

		go func() {
			icert.updates <- IstioCertUpdate{
				Op:        "update",
				Name:      secret.ObjectMeta.Name,
				Namespace: secret.ObjectMeta.Namespace,
				Secret:    secret,
			}

			dlog.Debugf(ctx, "%s: noted update!", icert)
		}()
	} else if deleted && !bothPresent {
		// OK, this is a deletion, and we no longer have both files. Time to
		// post the deletion.
		//
		// XXX We can actually generate _two_ deletions if both files are
		// deleted. We're not worrying about that for now.

		dlog.Debugf(ctx, "%s: noting deletion", icert)

		go func() {
			// Kind of a hack -- since we can't generate a real Secret object
			// without having the files we need, send the name & namespace from
			// icert.
			icert.updates <- IstioCertUpdate{
				Op:        "delete",
				Name:      icert.name,
				Namespace: icert.namespace,
				Secret:    nil,
			}

			dlog.Debugf(ctx, "%s: noted deletion!", icert)
		}()
	} else {
		dlog.Debugf(ctx, "%s: nothing to note", icert)
	}
}

// Secret generates a kates.Secret for this IstioCert. Since this
// involves reading PEM, it can fail, so it logs and returns a status.
func (icert *IstioCert) Secret(ctx context.Context) (*kates.Secret, bool) {
	privatePEM, privateOK := icert.readPEM(ctx, icert.dir, "key.pem")
	publicPEM, publicOK := icert.readPEM(ctx, icert.dir, "cert-chain.pem")

	if !privateOK || !publicOK {
		dlog.Errorf(ctx, "%s: read error, bailing", icert)
		return nil, false
	}

	newSecret := &kates.Secret{
		TypeMeta: kates.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      icert.name,
			Namespace: icert.namespace,
		},
		Type: kates.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": privatePEM,
			"tls.crt": publicPEM,
		},
	}

	return newSecret, true
}
